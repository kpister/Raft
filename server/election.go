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

// beginElection calls requestVote asynchronously to the other nodes
// it's simultaneously listening on votes response channel and done channel
// done cannel tells beginElection when to finish the election
// if the node got a majority of accepted or rejected votes, it ends the election by itself
func (n *node) beginElection(done chan string) bool {
	n.State = "candidate"
	n.CurrentTerm++

	log.Printf("election:%d\n", n.CurrentTerm)

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
			// election ends, return failed
			switch timeoutMsg {
			case "timeout":
				// timeout: remain candidate
				log.Println("election timeout")
				return false
			case "append entries newer":
				// append entries newer: become follower
				log.Println("election AE newer")
				return false
			}
		// got a repsond from runRequestVote
		case recvVote := <-voteChan:
			// haven't receive any vote from that node
			if !gotResp[n.ID] {
				log.Printf("recv vote:%d\n", n.ID)
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
				// got majority rejected votes
				// election ends
				if nResp-nGranted > len(n.ServersAddr) {
					// remain candidate
					return false
				}
			}
		}
	}
}

// runRequestVote sends requestVote to other nodes, and puts the result in voteChan
func (n *node) runRequestVote(nodeID int, voteChan chan vote) {
	req := &rf.RequestVoteRequest{
		Term:         n.CurrentTerm,
		CandidateId:  n.ID,
		LastLogIndex: n.CommitIndex,
		LastLogTerm:  n.Log[n.LastApplied].Term,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := n.ServersRaftClient[nodeID].RequestVote(ctx, req)
	n.errorHandler(err, "RequestVote", nodeID)

	if err == nil {
		voteChan <- vote{
			Granted: resp.VoteGranted,
			NodeID:  nodeID,
		}
	}
}

func (n *node) RequestVote(ctx context.Context, req *rf.RequestVoteRequest) (*rf.RequestVoteResponse, error) {
	log.Printf("got RequestVote:%d\n", req.CandidateId)

	resp := &rf.RequestVoteResponse{
		Term: n.CurrentTerm,
	}

	// 1. Reply false if term < currentTerm
	if req.Term < n.CurrentTerm {
		resp.VoteGranted = false
		log.Printf("resp RequestVote:%d reject smaller term\n", req.CandidateId)
		return resp, nil
	}

	// 2. If votedFor is null or candidateId, and candidate’s log is at least as up-to-date as receiver’s log, grant vote
	if (n.VotedFor == -1 || n.VotedFor == req.CandidateId) && (req.LastLogTerm > n.Log[n.LastApplied].Term || (req.LastLogTerm == n.Log[n.LastApplied].Term && req.LastLogIndex >= n.LastApplied)) {
		resp.VoteGranted = true
		log.Printf("resp RequestVote:%d accept\n", req.CandidateId)
		return resp, nil
	}

	resp.VoteGranted = false
	log.Printf("resp RequestVote:%d reject\n", req.CandidateId)
	return resp, nil
}
