package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	cm "github.com/kpister/raft/chaosmonkey"
	kv "github.com/kpister/raft/kvstore"
	raft "github.com/kpister/raft/raft"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type node struct {
	ID              int32
	Dict            map[string]string
	Chaos           [][]float32
	ServersAddr     []string
	ServersKvClient []kv.KeyValueStoreClient
	State           string

	// should be persistent
	CurrentTerm int32
	Log         []*raft.Entry // assume that first entry is dummy entry to make it 1 based indexing
	// hence len(log)-1 is number of entries in the log
	VotedFor int32

	// volatile
	LeaderID    int32
	CommitIndex int32
	LastApplied int32
}

func (n *node) Get(ctx context.Context, in *kv.GetRequest) (*kv.GetResponse, error) {
	log.Printf("GET:%s\n", in.Key)
	// get value @ key
	v, ok := n.Dict[in.Key]

	var r kv.ReturnCode
	if ok {
		log.Printf("GET success:%s\n", in.Key)
		r = kv.ReturnCode_SUCCESS
	} else {
		log.Printf("GET failed:%s\n", in.Key)
		r = kv.ReturnCode_FAILURE
	}

	return &kv.GetResponse{Value: v, Ret: r}, nil
}

func (n *node) Put(ctx context.Context, in *kv.PutRequest) (*kv.PutResponse, error) {
	log.Printf("PUT:%s %s\n", in.Key, in.Value)

	// put value to other replicates
	// from == -1 if the request is from client, not other replicates
	if in.From < 0 {
		n.Dict[in.Key] = in.Value

		for i := 0; i < len(n.ServersAddr); i++ {
			if i == int(n.ID) {
				continue
			}

			go func(idx int) {
				log.Printf("BC_PUT request:%s\n", n.ServersAddr[idx])
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				in.From = int32(n.ID)

				_, err := n.ServersKvClient[idx].Put(ctx, in)
				errStatus := status.Convert(err)
				switch errStatus.Code() {
				case codes.OK:
					log.Printf("BC_PUT success:%s\n", n.ServersAddr[idx])
					break
				case codes.Canceled:
					log.Printf("BC_PUT dropped:%s\n", n.ServersAddr[idx])
					break
				case codes.DeadlineExceeded:
					log.Printf("BC_PUT dropped:%s\n", n.ServersAddr[idx])
					break
				case codes.Unavailable:
					log.Printf("BC_PUT conn failed:%s\n", n.ServersAddr[idx])
					break
				default:
					log.Printf("BC_PUT failed:%s\n", n.ServersAddr[idx])
				}
			}(i)
		}
	} else {
		// decide if the message should be dropped
		r := rand.Float32()
		if r < n.Chaos[in.From][n.ID] {
			log.Printf("DROP_PUT:%f\n", n.Chaos[in.From][n.ID])
			time.Sleep(10 * time.Second)
		} else {
			log.Printf("BC_PUT:%s, %s\n", in.Key, in.Value)
			n.Dict[in.Key] = in.Value
		}
	}

	// set return code
	r := kv.ReturnCode_SUCCESS
	return &kv.PutResponse{Ret: r}, nil
}

func (n *node) UploadMatrix(ctx context.Context, mat *cm.ConnMatrix) (*cm.Status, error) {
	if len(mat.Rows) != len(n.ServersAddr) {
		return &cm.Status{Ret: cm.StatusCode_ERROR}, nil
	}

	for i := 0; i < len(mat.Rows); i++ {
		for j := 0; j < len(mat.Rows[i].Vals); j++ {
			if len(mat.Rows[i].Vals) != len(n.ServersAddr) {
				return &cm.Status{Ret: cm.StatusCode_ERROR}, nil
			}
			n.Chaos[i][j] = mat.Rows[i].Vals[j]
		}
	}
	log.Printf("UL_MAT\n")

	return &cm.Status{Ret: cm.StatusCode_OK}, nil
}

func (n *node) UpdateValue(ctx context.Context, matv *cm.MatValue) (*cm.Status, error) {
	if matv.Row < 0 || int(matv.Row) >= len(n.ServersAddr) || matv.Col < 0 || int(matv.Col) >= len(n.ServersAddr) {
		return &cm.Status{Ret: cm.StatusCode_ERROR}, nil
	}

	n.Chaos[matv.Row][matv.Col] = matv.Val
	log.Printf("UD_MAT:%d, %d, %f\n", matv.Row, matv.Col, matv.Val)

	return &cm.Status{Ret: cm.StatusCode_OK}, nil
}

func (n *node) AppendEntries(ctx context.Context, in *raft.AppendEntriesRequest) (*raft.AppendEntriesResponse, error) {

	response := &raft.AppendEntriesResponse{}

	// update the current state based on incoming things
	n.LeaderID = in.LeaderId

	// 1. Reply false if term < currentTerm (5.1)
	if in.Term < n.CurrentTerm {

		n.resetTimer("append entries newer")

		response.Success = false
		response.Reason = raft.ErrorCode_AE_OLDTERM
		response.Term = n.CurrentTerm
		return response, nil
	}

	n.resetTimer("append entries timer reset")

	// 2. reply false if log doesn't contain an entry at prevLogIndex whose term matches prevLogTerm

	if (int(in.PrevLogIndex) >= len(n.Log)) || (n.Log[in.PrevLogIndex].Term != in.PrevLogTerm) {
		response.Success = false
		response.Reason = raft.ErrorCode_AE_LOGMISMATCH
		response.Term = n.CurrentTerm
		return response, nil
	}

	// AT this point, we know that the leader's log and followers log match at PrevLogIndex
	// and hence by property, all previous log entries also match

	// Get new entries which leader sent
	leaderEntries := in.GetEntries()
	// 3. if an existing entry conflicts with a new one (same index but different terms),
	// delete the existing entry and all that follow it

	// we would delete every entry after in.PrevLogIndex before as anyways we have the new entries from leader
	n.Log = resizeSlice(n.Log, int(in.PrevLogIndex+1))
	// 4. append new entries not already present in the log
	n.Log = append(n.Log, leaderEntries...)

	// 5. if leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	indexOfLastNewEntry := len(n.Log) - 1
	if in.LeaderCommit > n.CommitIndex {
		n.CommitIndex = min(in.LeaderCommit, int32(indexOfLastNewEntry))
	}

	response.Success = true
	response.Reason = raft.ErrorCode_NONE
	response.Term = n.CurrentTerm
	return response, nil
}

func (n *node) ConnectServers() {
	for i := 0; i < len(n.ServersAddr); i++ {
		if i == int(n.ID) {
			continue
		}

		log.Printf("Connecting to %s\n", n.ServersAddr[i])
		conn, err := grpc.Dial(n.ServersAddr[i], grpc.WithInsecure())
		if err != nil {
			log.Printf("Failed to connect %s: %v\n", n.ServersAddr[i], err)
		}

		n.ServersKvClient[i] = kv.NewKeyValueStoreClient(conn)
	}
}

func (n *node) initialize() {
	netSize := len(n.ServersAddr)

	// initialize choasmonkey matrix with drop probability 0
	mat := make([][]float32, netSize)
	for i := 0; i < netSize; i++ {
		mat[i] = make([]float32, netSize)
		for j := 0; j < netSize; j++ {
			mat[i][j] = 0.0
		}
	}

	n.Chaos = mat
	n.Dict = make(map[string]string)
	n.ServersKvClient = make([]kv.KeyValueStoreClient, netSize)
	n.State = "follower"
	n.CurrentTerm = 0
}

var (
	server     node
	configFile = flag.String("config", "cfg.json", "the file to read the configuration from")
	help       = flag.Bool("h", false, "for usage")
)

func init() {
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		os.Exit(1)
	}

	readConfig(*configFile)
}

func readConfig(configFile string) {
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(configData, &server)
	server.initialize()

	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	// log setup
	f, err := os.OpenFile("log/"+time.Now().Format("2006.01.02_15:04:05.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Open log file error: %v\n", err)
	}
	defer f.Close()
	mw := io.MultiWriter(os.Stderr, f)
	log.SetOutput(mw)

	var opts []grpc.ServerOption

	lis, err := net.Listen("tcp", ":"+strings.Split(server.ServersAddr[server.ID], ":")[1])
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(opts...)
	kv.RegisterKeyValueStoreServer(grpcServer, &server)
	cm.RegisterChaosMonkeyServer(grpcServer, &server)
	server.ConnectServers()

	log.Printf("Listening on %s\n", server.ServersAddr[server.ID])
	grpcServer.Serve(lis)
}
