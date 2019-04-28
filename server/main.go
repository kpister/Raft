package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	cm "github.com/kpister/raft/chaosmonkey"
	kv "github.com/kpister/raft/kvstore"
	rf "github.com/kpister/raft/raft"
	"google.golang.org/grpc"
)

type node struct {
	ID                int32
	Dict              map[string]string
	Chaos             [][]float32
	ServersAddr       []string
	ServersKvClient   []kv.KeyValueStoreClient
	ServersRaftClient []rf.ServerClient
	State             string

	FollowerMax      int
	FollowerMin      int
	HeartbeatTimeout int

	// should be persistent
	CurrentTerm int32
	Log         []*rf.Entry // assume that first entry is dummy entry to make it 1 based indexing
	// hence len(log)-1 is number of entries in the log
	VotedFor int32

	// volatile
	LeaderID    int32
	CommitIndex int32
	LastApplied int32

	Timer      *time.Timer
	NextIndex  []int32
	MatchIndex []int32

	reset chan string
}

func (n *node) connectServers() {
	// grpc will retry in 15 ms at most 5 times when failed
	// TODO: put parameters into config
	opts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffLinear(time.Duration(15 * time.Millisecond))),
		grpc_retry.WithMax(5),
	}

	for i := 0; i < len(n.ServersAddr); i++ {
		if i == (int)(n.ID) {
			continue
		}

		log.Printf("Connecting to %s\n", n.ServersAddr[i])

		conn, err := grpc.Dial(n.ServersAddr[i], grpc.WithInsecure(),
			grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(opts...)))
		if err != nil {
			log.Printf("Failed to connect %s: %v\n", n.ServersAddr[i], err)
		}

		n.ServersKvClient[i] = kv.NewKeyValueStoreClient(conn)
		n.ServersRaftClient[i] = rf.NewServerClient(conn)
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
	n.ServersRaftClient = make([]rf.ServerClient, netSize)
	n.State = "follower"
	n.CurrentTerm = 0
	n.VotedFor = -1
	n.CommitIndex = 0
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
	server.connectServers()

	go server.loop()

	log.Printf("Listening on %s\n", server.ServersAddr[server.ID])
	grpcServer.Serve(lis)
}
