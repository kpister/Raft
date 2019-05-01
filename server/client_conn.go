package main

import (
	"context"
	"log"
	// "math/rand"
	"fmt"
	"strings"
	"time"

	kv "github.com/kpister/raft/kvstore"
	rf "github.com/kpister/raft/raft"
)

func (n *node) Put(ctx context.Context, in *kv.PutRequest) (*kv.PutResponse, error) {
	log.Printf("PUT:%s %s\n", in.Key, in.Value)

	// 1. reply NOT_LEADER if not leader, providing hint when available
	if n.LeaderID != n.ID {
		log.Printf("PUT FAILURE:NOT_LEADER:%d\n", n.LeaderID)
		return &kv.PutResponse{
			Ret:        kv.ReturnCode_FAILURE,
			LeaderHint: n.ServersAddr[n.LeaderID],
		}, nil
	}

	// appends the command to the log as new entry
	entry := &rf.Entry{
		Term:    n.CurrentTerm,
		Index:   (int32)(len(n.Log)),
		Command: fmt.Sprintf("%s$%s", in.Key, in.Value),
	}
	n.Log = append(n.Log, entry)

	// issues AppendEntries asynchronously here
	// collect responses from channel resps
	resps := make(chan rf.AppendEntriesResponse, len(n.ServersAddr))
	for i := 0; i < len(n.ServersAddr); i++ {
		if i == (int)(n.ID) {
			continue
		}

		go n.runAppendEntries(i, resps)
	}

	deadline, _ := ctx.Deadline()
	_, _, retCode := n.sendAppendEntries(deadline, resps, false)
	if retCode == kv.ReturnCode_FAILURE {
		log.Println("PUT FAILURE:AE")
	}
	return &kv.PutResponse{
		Ret: retCode,
	}, nil
}

func (n *node) sendAppendEntries(deadline time.Time, resps chan rf.AppendEntriesResponse, isGet bool) (int, int, kv.ReturnCode) {
	timer := time.NewTimer(time.Until(deadline))
	nSuccess := 1
	nResp := 1
	gotResp := make([]bool, len(n.ServersAddr))
	gotResp[n.ID] = true
	for {
		select {
		case val := <-resps:
			if !gotResp[val.Id] {
				gotResp[val.Id] = true

				if val.Success {
					nSuccess++
				} else if val.Reason == rf.ErrorCode_AE_OLDTERM {
					// we have been demoted, exit and become follower
					log.Println("REP FAIL:OLD_TERM")
					return nSuccess, nResp, kv.ReturnCode_FAILURE
				}
				nResp++

				if nSuccess > len(n.ServersAddr)/2 {
					// receive majority sucess responses
					log.Println("REP SUCC:MAJ_SUCC")
					return nSuccess, nResp, kv.ReturnCode_SUCCESS
				}

				if isGet && nResp > len(n.ServersAddr)/2 {
					// Get only, received majority responses
					log.Println("REP SUCC:MAJ_RESP")
					return nSuccess, nResp, kv.ReturnCode_SUCCESS
				}
			}
		case <-timer.C:
			log.Printf("REP SUCC:EXPIRED:%d/%d\n", nSuccess, nResp)
			return nSuccess, nResp, kv.ReturnCode_FAILURE
		}
	}
}

func (n *node) Get(ctx context.Context, in *kv.GetRequest) (*kv.GetResponse, error) {
	log.Printf("GET:%s\n", in.Key)

	// 1. check if it is a leader
	if n.LeaderID != n.ID {
		return &kv.GetResponse{
			Ret:        kv.ReturnCode_FAILURE,
			LeaderHint: n.ServersAddr[n.LeaderID],
		}, nil
	}

	// 2.
	for {
		if n.Log[n.CommitIndex].Term == n.CurrentTerm {
			break
		}
	}

	// 3.
	readIndex := n.CommitIndex

	// 4.
	// issues AppendEntries asynchronously here
	// collect responses from channel resps
	resps := make(chan rf.AppendEntriesResponse, len(n.ServersAddr))
	for i := 0; i < len(n.ServersAddr); i++ {
		if i == (int)(n.ID) {
			continue
		}

		go n.runAppendEntries(i, resps)
	}

	deadline, _ := ctx.Deadline()
	_, _, retCode := n.sendAppendEntries(deadline, resps, true)

	// 5.
	if retCode == kv.ReturnCode_SUCCESS {
		for i := readIndex; i >= 0; i-- {
			splits := strings.Split(n.Log[i].Command, "$")
			if splits[0] == in.Key {
				return &kv.GetResponse{
					Value: splits[1],
					Ret:   kv.ReturnCode_SUCCESS,
				}, nil
			}
		}
	}

	return &kv.GetResponse{
		Ret: kv.ReturnCode_FAILURE}, nil
}
