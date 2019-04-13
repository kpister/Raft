package main

import (
	"context"
	"fmt"
	"log"
	"net"

	kv "github.com/kpister/raft/kvstore"
	"google.golang.org/grpc"
)

type node struct {
	Dict map[string]string
}

func (n *node) Get(ctx context.Context, in *kv.GetRequest) (*kv.GetResponse, error) {
	fmt.Printf("Get Request recieved for key: %s\n", in.Key)
	// get value @ key
	//v := ...
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
	// set in.Key := in.Value
	n.Dict[in.Key] = in.Value

	// set return code
	r := kv.ReturnCode_SUCCESS
	return &kv.PutResponse{Ret: r}, nil
}

func newNode() *node {
	return &node{
		Dict: make(map[string]string),
	}
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
