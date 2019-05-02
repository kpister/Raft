package main

import (
	"log"
	"math/rand"
	"time"

	rf "github.com/kpister/raft/raft"
)

// getTimeout will return the per node timeout
// leader and follower timeouts are different
func (n *node) getTimeout() int {
	if n.State != "leader" {
		return n.FollowerMin + rand.Intn(n.FollowerMax-n.FollowerMin)
	} else {
		return n.HeartbeatTimeout
	}
}

// resetTimer will clear cause timeout to fire on reset
// it will set done <- message
func (n *node) resetTimer(message string) {
	select {
	case n.reset <- message:
	default:
		log.Println("Timer channel full")
	}
}

// timeout will fire whenever the timeout period occurs
// is reset by resetTimer, and by timeouts
func (n *node) timeout(done chan string) {
	for {
		select {
		// if reset received, set done message and restart
		case message := <-n.reset:
			// log.Printf("timeout:%s\n", message)
			select {
			case done <- message:
			default:
				log.Println("Done channel full")
			}
		// if the timer fired, return timeout
		case <-n.Timer.C:
			select {
			case done <- "timeout":
			default:
				log.Println("Done channel full")
			}
		}
		n.Timer.Reset(time.Duration(n.getTimeout()) * time.Millisecond)
	}
}

// initialize leader will promote a candidate to leader after
// a successful election cycle.
func (n *node) initializeLeader() {
	log.Println("candidate to leader")

	// For GET
	// As soon as this no-op entry is committed, the leader’s commit index will be at least as large as any other servers’ during its term
	n.Log = append(n.Log, &rf.Entry{
		Term:    n.CurrentTerm,
		Index:   (int32)(len(n.Log)),
		Command: "NOOP$NOOP",
	})

	log.Printf("commit index:%d\n", n.CommitIndex)
	for _, entry := range n.Log {
		log.Printf("%d %d %s\n", entry.Term, entry.Index, entry.Command)
	}

	n.NextIndex = make([]int32, len(n.ServersAddr))
	n.MatchIndex = make([]int32, len(n.ServersAddr))
	for i, _ := range n.ServersAddr {
		n.NextIndex[i] = (int32)(len(n.Log))
		n.MatchIndex[i] = 0
	}
	n.MatchIndex[n.ID] = int32(len(n.Log) - 1) // leader always match with its own entires

	// delim := "$"
	// apply the logs to the state machine
	// NOTE: Don't need this for now
	// for i := int(n.LastApplied) + 1; i < len(n.Log); i++ {
	// 	command := n.Log[i].Command
	// 	log.Printf("Applying the command::%s\n", command)
	// 	seperated := strings.Split(command, delim) // key{delim}value
	// 	if seperated[0] != "DUMMY" && seperated[0] != "NOOP" {
	// 		n.Dict[seperated[0]] = seperated[1]
	// 	}
	// }

	// NOTE: moved after the state machine application so that it can't serve client unless everything is applied
	n.State = "leader"
	n.LeaderID = n.ID

}

func (n *node) loop() {
	done := make(chan string, 1) // keep track of timeout and success
	n.Timer = time.NewTimer(time.Duration(n.getTimeout()) * time.Millisecond)

	// start a timer
	go n.timeout(done)

	// wait for timeout or response
	for {
		// followers will wait for timeouts, and respond to requests
		if n.State == "follower" {
			message := <-done
			// on timeout, become a candidate
			if message == "timeout" {
				n.State = "candidate"
			}
		} else if n.State == "candidate" { // candidates elect themselves, on win become a leader and heartbeat
			success := n.beginElection(done)
			if success {
				n.initializeLeader()
				n.heartbeat(done)
			}
		} else if n.State == "leader" {
			n.heartbeat(done)
		}
	}
}
