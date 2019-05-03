package main

import (
	"context"
	"log"
	"strconv"
	"time"

	cm "github.com/kpister/raft/chaosmonkey"
	"github.com/kpister/raft/client"
	kv "github.com/kpister/raft/kvstore"
)

// func getState(client *TYPE) {
// Get state from each client in list
// Store leader ID for commands (e.g. ISOLATE LEADER)
// Store parts of responses for ASSERTS
// }

// func ASSERT(no bool, statement *TYPE) {
// }

func getServerState(client cm.ChaosMonkeyClient) *cm.ServerState {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := client.GetState(ctx, &cm.EmptyMessage{})
	if err != nil {
		log.Fatalln("ERROR: GetState operation failed")
	}

	return res
}

func leaderFunctionality(serversAddr []string, clients []cm.ChaosMonkeyClient) {
	// ASSERT leader exists
	c := client.NewClient()
	c.SetClientID("testClient")
	cntLeader := 0
	leaderIndex := -1
	for i, servAddr := range serversAddr {
		c.Connect(servAddr)
		ret := c.MessagePut("key"+strconv.Itoa(i), "val"+strconv.Itoa(i))
		c.IncrementSeqNo()
		if ret == kv.ReturnCode_SUCCESS || ret == kv.ReturnCode_SUCCESS_SEQNO {
			cntLeader++
			leaderIndex = i
		}
	}
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
		if state.State == "follower" {
			for _, entry := range state.Log {
				// check if leader's no-op is in the follower's log
				if entry.Command == "NOOP$NOOP" && entry.Term == leaderTerm {
					noopCnt++
					break
				}
			}
		}
	}
	if noopCnt <= len(serversAddr)/2 {
		log.Fatalln("ERROR: leader's no-op is not recorded by a majority of followers")
	}

	log.Println("TEST PASSED")
}

func clientFunctionality(servAddr string) {
	// create client
	c := client.NewClient()
	c.SetClientID("testClient")
	c.Connect(servAddr)

	// ASSERT client.Put success
	firstKey, firstVal, secondVal := "key", "val_1", "val_2"
	ret := c.MessagePut(firstKey, firstVal)
	c.IncrementSeqNo()
	if ret != kv.ReturnCode_SUCCESS && ret != kv.ReturnCode_SUCCESS_SEQNO {
		log.Fatalln("ERROR: first PUT failed")
	}

	// -- allow retries or wait time
	time.Sleep(time.Duration(300 * time.Millisecond))

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

	// ASSERT client.Get receives the new value
	retVal, ret = c.MessageGet(firstKey)
	if ret != kv.ReturnCode_SUCCESS && ret != kv.ReturnCode_SUCCESS_SEQNO {
		log.Fatalln("ERROR: second GET failed")
	}
	if retVal != secondVal {
		log.Fatalf("ERROR: second GET response not matched: returned %s, expect %s\n", retVal, secondVal)
	}

	log.Println("TEST PASSED")
}

func logConsistency() {
	// ASSERT that ALL persistent logs are identical
	// -- allow for empty slots
	// ASSERT that ALL logs are identical up to CommitIndex
	// -- allow for empty slots
}
