package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	kv "github.com/kpister/raft/kvstore"
)

func (n *node) Put(ctx context.Context, in *kv.PutRequest) (*kv.PutResponse, error) {
	log.Printf("PUT:%s %s\n", in.Key, in.Value)

	// put value to other replicates
	// from == -1 if the request is from client, not other replicates
	if in.From < 0 {
		n.Dict[in.Key] = in.Value

		for i := 0; i < len(n.ServersAddr); i++ {
			if i == (int)(n.ID) {
				continue
			}

			go func(nodeID int) {
				log.Printf("BC_PUT request:%s\n", n.ServersAddr[nodeID])
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				in.From = int32(n.ID)

				_, err := n.ServersKvClient[nodeID].Put(ctx, in)
				n.errorHandler(err, "BC_PUT", nodeID)
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
