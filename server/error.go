package main

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
)

func (n *node) errorHandler(err error, task string, nodeID int) {
	errStatus := status.Convert(err)
	switch errStatus.Code() {
	case codes.OK:
		log.Printf("%s connected:%d\n", task, nodeID)
	case codes.Canceled:
		log.Printf("%s dropped:%d\n", task, nodeID)
	case codes.DeadlineExceeded:
		log.Printf("%s dropped:%d\n", task, nodeID)
	case codes.Unavailable:
		log.Printf("%s conn_failed:%d\n", task, nodeID)
	default:
		log.Printf("%s failed:%d\n", task, nodeID)
	}
}
