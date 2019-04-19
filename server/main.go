package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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
	ID              int
	Dict            map[string]string
	Chaos           [][]float32
	ServersAddr     []string
	ServersKvClient []kv.KeyValueStoreClient
	ServersCmClient []cm.ChaosMonkeyClient
}

func (n *node) Get(ctx context.Context, in *kv.GetRequest) (*kv.GetResponse, error) {
	fmt.Printf("Get Request recieved for key: %s\n", in.Key)
	// get value @ key
	v, ok := n.Dict[in.Key]

	var r kv.ReturnCode
	if ok {
		r = kv.ReturnCode_SUCCESS
	} else {
		r = kv.ReturnCode_FAILURE
	}

	return &kv.GetResponse{Value: v, Ret: r}, nil
}

func (n *node) Put(ctx context.Context, in *kv.PutRequest) (*kv.PutResponse, error) {
	fmt.Printf("Put Request recieved: set %s = %s\n", in.Key, in.Value)

	// put value to other replicates
	// from == -1 if the request is from client, not other replicates
	if in.From < 0 {
		n.Dict[in.Key] = in.Value

		for i := 0; i < len(n.ServersAddr); i++ {
			if i == n.ID {
				continue
			}

			go func(idx int) {
				log.Printf("Broadcast put request to %s\n", n.ServersAddr[idx])
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				in.From = int32(n.ID)

				res, err := n.ServersKvClient[idx].Put(ctx, in)
				errStatus := status.Convert(err)
				switch errStatus.Code() {
				case codes.OK:
					fmt.Printf("[%d] Status code: %d\n", idx, res.Ret)
					break
				case codes.Canceled:
					log.Printf("[%d] Put operation failed: the message was dropped\n", idx)
					break
				case codes.DeadlineExceeded:
					log.Printf("[%d] Put operation failed: the message was dropped\n", idx)
					break
				case codes.Unavailable:
					log.Printf("[%d] Put operation failed: Cannot connect to the replicate\n", idx)
					break
				default:
					log.Printf("[%d] Put operation failed: %v\n", idx, errStatus.Code())
				}
			}(i)
		}
	} else {
		// decide if the message should be dropped
		r := rand.Float32()
		log.Printf("random: %f, drop prob: %f\n", r, n.Chaos[in.From][n.ID])
		if r < n.Chaos[in.From][n.ID] {
			log.Printf("Message is dropped: %f\n", n.Chaos[in.From][n.ID])
			time.Sleep(10 * time.Second)
		} else {
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
		for i := 0; i < len(n.ServersAddr); i++ {
			if i == n.ID {
				continue
			}

			log.Printf("Broadcast upload matrix to %s\n", n.ServersAddr[i])
			mat.From = int32(n.ID)

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
		for i := 0; i < len(n.ServersAddr); i++ {
			if i == n.ID {
				continue
			}

			log.Printf("Broadcast update matrix value to %s\n", n.ServersAddr[i])
			matv.From = int32(n.ID)

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
	for i := 0; i < len(n.ServersAddr); i++ {
		if i == n.ID {
			continue
		}

		conn, err := grpc.Dial(n.ServersAddr[i], grpc.WithInsecure())
		if err != nil {
			log.Fatalf("Failed to connect: %v\n", err)
		}

		n.ServersKvClient[i] = kv.NewKeyValueStoreClient(conn)
		n.ServersCmClient[i] = cm.NewChaosMonkeyClient(conn)
	}
}

func (n *node) initialize() {
	net_size := len(n.ServersAddr)

	// initialize choasmonkey matrix with drop probability 0
	mat := make([][]float32, net_size)
	for i := 0; i < net_size; i++ {
		mat[i] = make([]float32, net_size)
		for j := 0; j < net_size; j++ {
			mat[i][j] = 0.0
		}
	}

	n.Chaos = mat
	n.Dict = make(map[string]string)
	n.ServersKvClient = make([]kv.KeyValueStoreClient, net_size)
	n.ServersCmClient = make([]cm.ChaosMonkeyClient, net_size)
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
