package main

import (
	"context"
	"log"
	"time"

	rf "github.com/kpister/raft/raft"
)

// heartbeat will send a message to every other node,
// it attempts to bring those nodes up to speed
// asynchronously starts many runAppendEntries threads
func (n *node) heartbeat(done chan string) {
	// log.Printf("heartbeat:%d\n", n.CurrentTerm)

	responses := make(chan rf.AppendEntriesResponse, len(n.ServersAddr))

	for i, _ := range n.ServersAddr {
		// skip self
		if i == (int)(n.ID) {
			continue
		}

		// responses stores each go routine value
		go n.runAppendEntries(i, responses)
	}

	// collect all responses as they come in
	for {
		select {
		case val := <-responses:
			// on success, we update the match index to the point we found a match
			// then reset nextIndex so the next heartbeat will match farther
			if val.Success {
				n.MatchIndex[val.Id] = n.NextIndex[val.Id] - 1
				n.NextIndex[val.Id] = (int32)(len(n.Log))
			} else if val.Reason == rf.ErrorCode_AE_OLDTERM { // we have been demoted, exit and become follower
				n.State = "follower"
				n.CurrentTerm = val.Term
				n.resetTimer("AE term mismatch")
				return
			} else if val.Reason == rf.ErrorCode_AE_LOGMISMATCH { // this node ID is out of date, attempt to find earliest matching point
				n.NextIndex[val.Id]--
			}

			n.updateCommitIndex()
		// check for timeout, if so, exit and restart
		case message := <-done:
			if message == "timeout" {
				return
			}
		}
	}
}

// AppendEntries is called by the leader to update the logs and refresh timeout
func (n *node) AppendEntries(ctx context.Context, in *rf.AppendEntriesRequest) (*rf.AppendEntriesResponse, error) {

	response := &rf.AppendEntriesResponse{}

	// update the current state based on incoming things
	n.LeaderID = in.LeaderId
	n.CurrentTerm = max(n.CurrentTerm, in.Term)

	// 1. Reply false if term < currentTerm (5.1)
	if in.Term < n.CurrentTerm {
		response.Success = false
		response.Reason = rf.ErrorCode_AE_OLDTERM
		response.Term = n.CurrentTerm
		return response, nil
	}

	// receive request with term >= currentTerm
	// become follower (it could be candidate or follower)
	if n.State == "candidate" {
		log.Println("AE newer term:candidate to follower")
	}
	n.State = "follower"
	n.resetTimer("append entries newer term")

	// 2. reply false if log doesn't contain an entry at prevLogIndex whose term matches prevLogTerm

	if (int(in.PrevLogIndex) >= len(n.Log)) || (n.Log[in.PrevLogIndex].Term != in.PrevLogTerm) {
		response.Success = false
		response.Reason = rf.ErrorCode_AE_LOGMISMATCH
		response.Term = n.CurrentTerm
		return response, nil
	}

	// AT this point, we know that the leader's log and followers log match at PrevLogIndex
	// and hence by property, all previous log entries also match

	// Get new entries which leader sent
	leaderEntries := in.GetEntries()
	// 3. if an existing entry conflicts with a new one (same index but different terms),
	// delete the existing entry and all that follow it

	// we would delete every entry after in.PrevLogIndex before as anyways we have the new entries from leader
	n.Log = resizeSlice(n.Log, int(in.PrevLogIndex+1))
	// 4. append new entries not already present in the log
	n.Log = append(n.Log, leaderEntries...)

	// 5. if leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	indexOfLastNewEntry := len(n.Log) - 1
	if in.LeaderCommit > n.CommitIndex {
		n.CommitIndex = min(in.LeaderCommit, int32(indexOfLastNewEntry))
	}

	// PERSIST LOG before returning with success
	if len(leaderEntries) > 0 {
		n.persistLog()
	}

	response.Success = true
	response.Reason = rf.ErrorCode_NONE
	response.Term = n.CurrentTerm
	return response, nil
}

func (n *node) runAppendEntries(node_id int, resp chan rf.AppendEntriesResponse) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(n.HeartbeatTimeout)*time.Millisecond)
	defer cancel()

	var entries []*rf.Entry
	if (int)(n.NextIndex[node_id]) == len(n.Log) {
		entries = make([]*rf.Entry, 0)
	} else {
		entries = n.Log[n.NextIndex[node_id]-1:]
	}

	args := rf.AppendEntriesRequest{
		Term:         n.CurrentTerm,
		LeaderId:     n.ID,
		PrevLogIndex: n.NextIndex[node_id] - 1,
		PrevLogTerm:  n.Log[n.NextIndex[node_id]-1].Term,
		LeaderCommit: n.CommitIndex,
		Entries:      entries,
	}

	r, err := n.ServersRaftClient[node_id].AppendEntries(ctx, &args)
	if err != nil {
		n.errorHandler(err, "AE", node_id)
	} else {
		resp <- *r
	}
}

// updateCommitIndex finds the largest index with a majority match
func (n *node) updateCommitIndex() {
	// start at commitindex as it won't be worse than that
	for i := n.CommitIndex + 1; i < (int32)(len(n.Log)); i++ {
		count := 0
		for _, val := range n.MatchIndex {
			if val >= i {
				count += 1
			}
		}
		if count >= len(n.ServersAddr)/2+1 {
			n.CommitIndex = i
		}
	}
}
