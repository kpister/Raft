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
		log.Printf("%s success:%s\n", task, n.ServersAddr[nodeID])
		break
	case codes.Canceled:
		log.Printf("%s dropped:%s\n", task, n.ServersAddr[nodeID])
		break
	case codes.DeadlineExceeded:
		log.Printf("%s dropped:%s\n", task, n.ServersAddr[nodeID])
		break
	case codes.Unavailable:
		log.Printf("%s conn failed:%s\n", task, n.ServersAddr[nodeID])
		break
	default:
		log.Printf("%s failed:%s\n", task, n.ServersAddr[nodeID])
	}
}
