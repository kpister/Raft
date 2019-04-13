package main

import (
    "log"
    "fmt"
    "net"
    "context"

    "google.golang.org/grpc"
    kv "github.com/kpister/raft/kvstore"
)

type node struct{}

func (n *node) Get(ctx context.Context, in *kv.GetRequest) (*kv.GetResponse, error) {
    fmt.Printf("Get Request recieved for key: %s", in.Key)
    // get value @ key
    //v := ...
    v := "1"

    // set return code
    r := kv.ReturnCode_SUCCESS
    return &kv.GetResponse{Value: v, Ret: r}, nil
}

func (n *node) Put(ctx context.Context, in *kv.PutRequest) (*kv.PutResponse, error) {
    fmt.Printf("Put Request recieved: set %s = %d", in.Key, in.Value)
    // set in.Key := in.Value

    // set return code
    r := kv.ReturnCode_SUCCESS
    return &kv.PutResponse{Ret: r}, nil
}

func newNode() *node {
    return &node{}
}

const (
    port = 5000
)

func main() {
    var opts []grpc.ServerOption

    lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    grpcServer := grpc.NewServer(opts...)
    kv.RegisterKeyValueStoreServer(grpcServer, newNode())
    grpcServer.Serve(lis)
}
