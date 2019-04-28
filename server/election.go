package main

import (
	"context"
	"log"
	"time"

	rf "github.com/kpister/raft/raft"
)

type vote struct {
	Granted bool
	NodeID  int
}

func (n *node) beginElection(done chan string) bool {
	n.State = "candidate"
	n.CurrentTerm++

	// send requestVote to all the other nodes
	// collect votes from voteChan
	voteChan := make(chan vote, len(n.ServersAddr))
	for i := 0; i < len(n.ServersAddr); i++ {
		if i == (int)(n.ID) {
			continue
		}

		go n.runRequestVote(i, voteChan)
	}

	// vote for itself first
	nGranted := 1
	nResp := 1
	gotResp := make([]bool, len(n.ServersAddr))
	gotResp[n.ID] = true
	for {
		select {
		// got a timeout message from other goroutine (timer timeout, appendEnries, etc.)
		case timeoutMsg := <-done:
			// timeout before election ends
			// election ends
			switch timeoutMsg {
			case "timeout":
				// remain candidate
				return false
			case "append entries newer":
				// other node got more up-to-date log
				// become follower
				return false
			default:
				// else
			}
		// got a repsond from runRequestVote
		case recvVote := <-voteChan:
			// haven't receive any vote from that node
			if !gotResp[n.ID] {
				gotResp[n.ID] = true

				if recvVote.Granted {
					nGranted++
				}
				nResp++

				// got majority granted votes
				// win the election
				if nGranted > len(n.ServersAddr)/2 {
					// become leader
					return true
				}
				// got majority against votes
				// election ends
				if nResp-nGranted > len(n.ServersAddr) {
					// remain candidate
					return false
				}
			}
		}
	}
}

func (n *node) runRequestVote(nodeID int, voteChan chan vote) {
	req := &rf.RequestVoteRequest{
		Term:         n.CurrentTerm,
		CandidateId:  n.ID,
		LastLogIndex: n.CommitIndex,
		// LastLogTerm
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := n.ServersRaftClient[nodeID].RequestVote(ctx, req)
	if err != nil {
		log.Printf("RequestVote error: %v\n", err)
	}

	voteChan <- vote{
		Granted: resp.VoteGranted,
		NodeID:  nodeID,
	}
}

func (n *node) RequestVote(ctx context.Context, req *rf.RequestVoteRequest) (*rf.RequestVoteResponse, error) {
	resp := &rf.RequestVoteResponse{
		Term: n.CurrentTerm,
	}

	if req.Term < n.CurrentTerm {
		resp.VoteGranted = false
		return resp, nil
	}

	if (n.VotedFor == -1 || n.VotedFor == req.CandidateId) && (req.LastLogTerm > n.CurrentTerm || (req.LastLogTerm == n.CurrentTerm && req.LastLogTerm > n.CommitIndex)) {
		resp.VoteGranted = true
		return resp, nil
	}

	resp.VoteGranted = false
	return resp, nil
}
