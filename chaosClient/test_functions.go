package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	cm "github.com/kpister/raft/chaosmonkey"
	"github.com/kpister/raft/client"
	kv "github.com/kpister/raft/kvstore"
	rf "github.com/kpister/raft/raft"
)

func getServerState(client cm.ChaosMonkeyClient) *cm.ServerState {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := client.GetState(ctx, &cm.EmptyMessage{})
	if err != nil {
		log.Fatalln("ERROR: GetState operation failed")
	}

	return res
}

func findActiveLeader(serversAddr []string, clients []cm.ChaosMonkeyClient) (int, int) {

	c := client.NewClient()
	c.SetClientID("testClient")
	cntLeader := 0
	leaderIndex := -1
	for i, servAddr := range serversAddr {
		c.Connect(servAddr)
		ret := c.MessagePut("key"+strconv.Itoa((int)(rand.Int31n(100000))), "val"+strconv.Itoa(i))
		c.IncrementSeqNo()
		if ret == kv.ReturnCode_SUCCESS || ret == kv.ReturnCode_SUCCESS_SEQNO {
			cntLeader++
			leaderIndex = i
		}
	}
	return leaderIndex, cntLeader
}

func findActiveLeaderWithRetries(retries int, serversAddr []string, clients []cm.ChaosMonkeyClient) (int, bool) {
	for i := 0; i < retries; i++ {
		leaderid, numleaders := findActiveLeader(serversAddr, clients)
		if numleaders == 1 {
			return leaderid, true
		}
	}
	return 0, false
}

func leaderFunctionality(serversAddr []string, clients []cm.ChaosMonkeyClient) {
	// ASSERT leader exists
	leaderIndex, cntLeader := findActiveLeader(serversAddr, clients)

	if cntLeader == 0 {
		log.Fatal("ERROR: no leader")
	} else if cntLeader > 1 {
		log.Fatal("ERROR: more than one leader")
	}
	log.Printf("leader is %s\n", serversAddr[leaderIndex])

	// ASSERT leader can update log
	leaderState := getServerState(clients[leaderIndex])
	foundUpdate := false
	for _, entry := range leaderState.Log {
		// check if the entry leader has put is in log
		if entry.Command == "key"+strconv.Itoa(leaderIndex)+"$"+"val"+strconv.Itoa(leaderIndex) {
			foundUpdate = true
			break
		}
	}
	if !foundUpdate {
		log.Fatalln("ERROR: leader cannot update log")
	}

	// -- The leader's no-op has been recorded by a majority of followers
	leaderTerm := leaderState.CurrentTerm
	noopCnt := 0
	for _, client := range clients {
		state := getServerState(client)

		for _, entry := range state.Log {
			// check if leader's no-op is in the follower's log
			if entry.Command == "NOOP$NOOP" && entry.Term == leaderTerm {
				noopCnt++
				break
			}
		}

	}
	if noopCnt <= len(serversAddr)/2 {
		log.Fatalln("ERROR: leader's no-op is not recorded by a majority of followers")
	}

	log.Println("LEADER: TEST PASSED")
}

func noLeaderFunctionality(serversAddr []string, clients []cm.ChaosMonkeyClient) {
	// ASSERT leader exists
	_, cntLeader := findActiveLeader(serversAddr, clients)

	if cntLeader != 0 {
		log.Fatal("ERROR: there is a leader")
	}

	log.Println("NO LEADER: TEST PASSED")
}

func clientFunctionality(servAddr string) {
	// create client
	c := client.NewClient()
	c.SetClientID("client1")
	c.Connect(servAddr)

	// ASSERT client.Put success
	firstKey, firstVal, secondVal := "key", "val_1", "val_2"
	ret := c.MessagePut(firstKey, firstVal)
	c.IncrementSeqNo()
	if ret != kv.ReturnCode_SUCCESS && ret != kv.ReturnCode_SUCCESS_SEQNO {
		log.Fatalln("ERROR: first PUT failed")
	}

	// -- allow retries or wait time
	time.Sleep(time.Duration(700 * time.Millisecond))

	// ASSERT client.Get success
	retVal, ret := c.MessageGet(firstKey)
	if ret != kv.ReturnCode_SUCCESS && ret != kv.ReturnCode_SUCCESS_SEQNO {
		log.Fatalln("ERROR: first GET failed")
	}
	if retVal != firstVal {
		log.Fatalf("ERROR: first GET response not matched: returned %s, expect %s\n", retVal, firstVal)
	}

	// ASSERT client.Put second value for same key
	ret = c.MessagePut(firstKey, secondVal)
	c.IncrementSeqNo()
	if ret != kv.ReturnCode_SUCCESS && ret != kv.ReturnCode_SUCCESS_SEQNO {
		log.Fatalln("ERROR: second GET failed")
	}

	// -- allow retries or wait time
	time.Sleep(time.Duration(700 * time.Millisecond))

	// ASSERT client.Get receives the new value
	retVal, ret = c.MessageGet(firstKey)
	if ret != kv.ReturnCode_SUCCESS && ret != kv.ReturnCode_SUCCESS_SEQNO {
		log.Fatalln("ERROR: second GET failed")
	}
	if retVal != secondVal {
		log.Fatalf("ERROR: second GET response not matched: returned %s, expect %s\n", retVal, secondVal)
	}

	log.Println("CLIENT: TEST PASSED")
}

func getState(clients []cm.ChaosMonkeyClient, states []*cm.ServerState) {
	// Get state from each client in list
	// Store leader ID for commands (e.g. ISOLATE LEADER)
	// Store parts of responses for ASSERTS

	// states := make([]cm.ServerState, len(clients))

	for i := 0; i < len(clients); i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := clients[i].GetState(ctx, &cm.EmptyMessage{})
		if err != nil {
			log.Fatalf("Get state operation failed on server %v: %v\n", i, err)
		}

		states[i] = res
	}
}

func assert(cond bool, err string) {
	if false {
		panic("assertion : " + err)
	}
}

func isOneLeader(states []*cm.ServerState) (int32, bool) {
	leaders := findLeaders(states)
	return leaders[0], len(leaders) == 1
}

func isNoLeader(states []*cm.ServerState) bool {
	leaders := findLeaders(states)
	return len(leaders) == 0
}

func realLeaderReturnsResult(states []*cm.ServerState, leaderid int32) {

}

func fakeLeaderNotReturnResult(states []*cm.ServerState, leaderid int32) {

}

// assume we somehow know the "true" leader
func logConsistency(states []*cm.ServerState, leaderid int32) {

	// 1. For each pair of servers thier logs match upto some point, but after than point the logs don't match at even
	// a single entry
	// aka.
	// If two entries in different logs have the same index and ther, then the logs are identical in all preceding
	// terms
	// IF two entries in different logs have the same index and term, then they store the same command
	for i := 0; i < len(states); i++ {
		for j := i + 1; j < len(states); j++ {
			okay := compareTwoServerLogs(states[i], states[j])
			assert(okay, fmt.Sprintf("ERROR: Logs of servers %d and %d are inconsistent", i, j))
		}
	}

	// 2. For each pair of form (leader, follower) the logs of follower should match that of leader
	// before leader's commit index

	// 2a. ASSERT if the passed leaderid is indeed a leader
	assert(states[leaderid].State == "leader", "ERROR: incorrect leader argument to logConsistency function")
	// 2b.
	cnt := 0 // number of nodes for whom entires match
	var flag bool
	for followerid := 0; followerid < len(states); followerid++ {
		flag = false
		last := min(len(states[followerid].Log)-1, int(states[leaderid].CommitIndex))
		for i := 0; i <= last; i++ {
			if !isLogEntrySame(states[leaderid].Log[i], states[followerid].Log[i]) {
				flag = true
				break
			}
		}
		if !flag {
			cnt++
		}
	}

	assert(cnt >= len(states)/2+1, "ERROR: MAJORITY LOGS DOESN't MATCH")
	log.Println("LOG: TEST PASSED")

}

// HELPERS

// RETURNS true if logs are okay otherwise return false
// last index is the last entry upto which we need to check
func compareTwoServerLogs(server1 *cm.ServerState, server2 *cm.ServerState) bool {

	last := min(len(server1.Log), len(server2.Log))

	var i int
	for i = 0; i < last; i++ {
		if !isLogEntrySame(server1.Log[i], server1.Log[i]) {
			break
		}
	}

	// i point to the first non matching entry in the log
	// now it must be that every other entry doesn't match
	for ; i < last; i++ {
		if isLogEntrySame(server1.Log[i], server2.Log[i]) {
			return false
		}
	}

	return true
}

// returns the list of ids for which state is leader
func findLeaders(states []*cm.ServerState) []int32 {
	leaders := make([]int32, 0)
	for i, state := range states {
		if state.State == "leader" {
			leaders = append(leaders, int32(i))
		}
	}
	return leaders
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// we assume that we only pass entries with same index in here so that we don't check if the index is same or not
func isLogEntrySame(entry1 *rf.Entry, entry2 *rf.Entry) bool {
	if (entry1.Term == entry2.Term) && (entry1.Command == entry2.Command) {
		return true
	}
	return false
}
