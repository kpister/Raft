package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	cm "github.com/kpister/raft/chaosmonkey"
	kv "github.com/kpister/raft/kvstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type node struct {
	Dict            map[string]string
	NServers        int
	Chaos           [][]float32
	ServersAddr     []string
	ServerIndex     int
	ServersKvClient []kv.KeyValueStoreClient
	ServersCmClient []cm.ChaosMonkeyClient
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
	fmt.Printf("PUT:%s, %s\n", in.Key, in.Value)

	// put value to other replicates
	// from == -1 if the request is from client, not other replicates
	if in.From < 0 {
		n.Dict[in.Key] = in.Value

		for i := 0; i < n.NServers; i++ {
			if i == n.ServerIndex {
				continue
			}

			go func(idx int) {
				log.Printf("BROADCAST request:%s\n", n.ServersAddr[idx])
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				in.From = int32(n.ServerIndex)

				_, err := n.ServersKvClient[idx].Put(ctx, in)
				errStatus := status.Convert(err)
				switch errStatus.Code() {
				case codes.OK:
					fmt.Printf("BROADCAST_PUT success:%s\n", n.ServersAddr[idx])
					break
				case codes.Canceled:
					log.Printf("BROADCAST_PUT dropped:%s\n", n.ServersAddr[idx])
					break
				case codes.DeadlineExceeded:
					log.Printf("BROADCAST_PUT dropped:%s\n", n.ServersAddr[idx])
					break
				case codes.Unavailable:
					log.Printf("BROADCAST_PUT conn failed:%s\n", n.ServersAddr[idx])
					break
				default:
					log.Printf("BROADCAST_PUT failed:%s\n", n.ServersAddr[idx])
				}
			}(i)
		}
	} else {
		// decide if the message should be dropped
		r := rand.Float32()
		if r < n.Chaos[in.From][n.ServerIndex] {
			log.Printf("DROP_PUT:%f\n", n.Chaos[in.From][n.ServerIndex])
			time.Sleep(10 * time.Second)
		} else {
			fmt.Printf("PUT_BROADCAST:%s, %s\n", in.Key, in.Value)
			n.Dict[in.Key] = in.Value
		}
	}

	// set return code
	r := kv.ReturnCode_SUCCESS
	return &kv.PutResponse{Ret: r}, nil
}

func (n *node) UploadMatrix(ctx context.Context, mat *cm.ConnMatrix) (*cm.Status, error) {
	for i := 0; i < len(mat.Rows); i++ {
		for j := 0; j < len(mat.Rows[i].Vals); j++ {
			n.Chaos[i][j] = mat.Rows[i].Vals[j]
		}
	}
	fmt.Println(n.Chaos)

	if mat.From < 0 {
		for i := 0; i < n.NServers; i++ {
			if i == n.ServerIndex {
				continue
			}

			log.Printf("Broadcast upload matrix to %s\n", n.ServersAddr[i])
			mat.From = int32(n.ServerIndex)

			res, err := n.ServersCmClient[i].UploadMatrix(ctx, mat)
			if err != nil {
				log.Printf("Upload matrix failed: %v\n", err)
			} else {
				log.Printf("Status code: %d\n", res.Ret)
			}
		}
	}

	return &cm.Status{Ret: cm.StatusCode_OK}, nil
}

func (n *node) UpdateValue(ctx context.Context, matv *cm.MatValue) (*cm.Status, error) {
	n.Chaos[matv.Row][matv.Col] = matv.Val
	fmt.Println(n.Chaos)

	if matv.From < 0 {
		for i := 0; i < n.NServers; i++ {
			if i == n.ServerIndex {
				continue
			}

			log.Printf("Broadcast update matrix value to %s\n", n.ServersAddr[i])
			matv.From = int32(n.ServerIndex)

			res, err := n.ServersCmClient[i].UpdateValue(ctx, matv)
			if err != nil {
				log.Printf("Update value failed: %v\n", err)
			} else {
				log.Printf("Status code: %d\n", res.Ret)
			}
		}
	}

	return &cm.Status{Ret: cm.StatusCode_OK}, nil
}

func (n *node) ConnectServers() {
	for i := 0; i < n.NServers; i++ {
		if i == n.ServerIndex {
			continue
		}

		log.Printf("Connecting to %s\n", n.ServersAddr[i])
		conn, err := grpc.Dial(n.ServersAddr[i], grpc.WithInsecure())
		if err != nil {
			log.Printf("Failed to connect %s: %v\n", n.ServersAddr[i], err)
		}

		n.ServersKvClient[i] = kv.NewKeyValueStoreClient(conn)
		n.ServersCmClient[i] = cm.NewChaosMonkeyClient(conn)
	}
}

func newNode(serversAddr []string, serverIndex int) *node {
	n := len(serversAddr)

	// initialize choasmonkey matrix with drop probability 0
	mat := make([][]float32, n)
	for i := 0; i < n; i++ {
		mat[i] = make([]float32, n)
		for j := 0; j < n; j++ {
			mat[i][j] = 0.0
		}
	}

	return &node{
		Dict:            make(map[string]string),
		NServers:        n,
		Chaos:           mat,
		ServersAddr:     serversAddr,
		ServerIndex:     serverIndex,
		ServersKvClient: make([]kv.KeyValueStoreClient, n),
		ServersCmClient: make([]cm.ChaosMonkeyClient, n),
	}
}

func readConfig(configPath string) []string {
	configFp, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("Open config file error: %v\n", err)
	}
	defer configFp.Close()

	scanner := bufio.NewScanner(configFp)
	var addrs []string
	for scanner.Scan() {
		addr := strings.TrimSpace(scanner.Text())
		addrs = append(addrs, addr)
	}

	return addrs
}

var configPath string
var serverIndex int

func init() {
	flag.StringVar(&configPath, "config", "config.txt", "path of the config file")
	flag.IntVar(&serverIndex, "srv_idx", 0, "index of the server")
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

	flag.Parse()

	serversAddr := readConfig(configPath)

	lis, err := net.Listen("tcp", serversAddr[serverIndex])
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(opts...)
	node := newNode(serversAddr, serverIndex)
	kv.RegisterKeyValueStoreServer(grpcServer, node)
	cm.RegisterChaosMonkeyServer(grpcServer, node)
	node.ConnectServers()

	log.Printf("Listening on %s\n", node.ServersAddr[node.ServerIndex])
	grpcServer.Serve(lis)
}
