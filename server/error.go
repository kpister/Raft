package main

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
)

func (n *node) errorHandler(err error, task string, nodeID int) string {
	errStatus := status.Convert(err)
	switch errStatus.Code() {
	case codes.OK:
		log.Printf("%s connected:%d\n", task, nodeID)
		return "conn"
	case codes.Canceled:
		log.Printf("%s dropped:%d\n", task, nodeID)
		return "dropped"
	case codes.DeadlineExceeded:
		log.Printf("%s dropped:%d\n", task, nodeID)
		return "dropped"
	case codes.Unavailable:
		log.Printf("%s conn_failed:%d\n", task, nodeID)
		return "conn_failed"
	default:
		log.Printf("%s failed:%d\n", task, nodeID)
		return "failed"
	}
}
