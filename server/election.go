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
	n.VotedFor = -1

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
				log.Println("election:timeout")
				return false
			case "append entries newer term":
				// append entries newer term: become follower
				log.Println("election:AE newer")
				return false
			}
		// got a repsond from runRequestVote
		case recvVote := <-voteChan:
			// haven't receive any vote from that node
			if !gotResp[recvVote.NodeID] {
				gotResp[n.ID] = true

				if recvVote.Granted {
					log.Printf("recv accept vote:%d\n", recvVote.NodeID)
					nGranted++
				} else {
					log.Printf("recv reject vote:%d\n", recvVote.NodeID)
				}
				nResp++

				// got majority granted votes
				// win the election
				if nGranted > len(n.ServersAddr)/2 {
					// become leader
					log.Printf("election end:got majority accept")
					return true
				}
				// got majority rejected votes
				// election ends
				if nResp-nGranted > len(n.ServersAddr) {
					// remain candidate
					log.Printf("election end:got majority reject")
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
	if err != nil {
		n.errorHandler(err, "RequestVote", nodeID)
	} else {
		voteChan <- vote{
			Granted: resp.VoteGranted,
			NodeID:  nodeID,
		}
	}
}

func (n *node) RequestVote(ctx context.Context, req *rf.RequestVoteRequest) (*rf.RequestVoteResponse, error) {
	// log.Printf("%d:RequestVote FROM:%d, TERM:%d\n", n.ID, req.CandidateId, req.Term)

	resp := &rf.RequestVoteResponse{
		Term: n.CurrentTerm,
	}

	// CHAOS monkey part
	shouldDrop := n.dropMessageChaos(req.CandidateId)
	if shouldDrop {
		// behavior of what to do when dropping the message
		log.Printf("%d:DROPPING: Request Vote from %d\n", n.ID, req.CandidateId)
		time.Sleep(20 * time.Second)
		// just for safety
		// in reality 20 seconds should always be greateer than the context deadline and this should never return
		// anything
		resp.VoteGranted = false
		return resp, nil
	}

	// 1. Reply false if term < currentTerm
	if req.Term < n.CurrentTerm {
		resp.VoteGranted = false
		log.Printf("resp RequestVote: FROM %d: TERM:%d, MyTERM: %d: reject smaller term\n", req.CandidateId, req.Term, n.CurrentTerm)
		return resp, nil
	}

	// 2. If votedFor is null or candidateId, and candidate’s log is at least as up-to-date as receiver’s log, grant vote
	if (n.VotedFor == -1 || n.VotedFor == req.CandidateId) && (req.LastLogTerm > n.Log[n.LastApplied].Term || (req.LastLogTerm == n.Log[n.LastApplied].Term && req.LastLogIndex >= n.LastApplied)) {
		resp.VoteGranted = true
		n.VotedFor = req.CandidateId
		log.Printf("resp RequestVote: FROM %d: TERM:%d, MyTERM: %d: accept\n", req.CandidateId, req.Term, n.CurrentTerm)
		return resp, nil
	}

	resp.VoteGranted = false
	log.Printf("resp RequestVote: FROM %d: TERM:%d, MyTERM: %d: reject\n", req.CandidateId, req.Term, n.CurrentTerm)
	return resp, nil
}
