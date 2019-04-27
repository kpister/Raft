package main

import "log"
import "time"
import "math/rand"
import "github.com/kpister/raft/raft"

// getTimeout will return the per node timeout
// leader and follower timeouts are different
func (n *node) getTimeout() int {
    if n.State != "leader" {
        return n.FollowerMin + rand.Intn(n.FollowerMax - n.FollowerMin)
    } else {
        return n.HeartbeatTimeout
    }
}

// resetTimer will clear cause timeout to fire on reset
// it will set done <- message
func (n *node) resetTimer(message string) {
    select{
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
            select {
            case done <- message:
            default:
                log.Println("Done channel full")
            }
        // if the timer fired, return timeout
        case <-timer.C:
            select {
            case done <- "timeout":
            default:
                log.Println("Done channel full")
            }
        }
        n.timer.Reset(n.getTimeout() * time.Millisecond)
    }
}

func (n *node) updateCommitIndex() {
    // find the largest index with a majority match
    best := n.CommitIndex
    for i := n.CommitIndex+1; i < len(n.Log); i++ {
        count := 0
        for _, val := range n.MatchIndex {
            if val >= i {
                count += 1
            }
        }
        if count >= len(n.ServersAddr) / 2 + 1 {
            best = i
        }
    }
    n.CommitIndex = best
}

// heartbeat will send a message to every other node,
// it attempts to bring those nodes up to speed
// asynchronously starts many runAppendEntries threads
func (n *node) heartbeat(done chan string) {
    responses := make(chan raft.AppendEntriesResponse, len(n.ServersAddr))

    for i, node := range n.ServersAddr {
        // skip self
        if i == n.ID {
            continue
        }

        // responses stores each go routine value
        go n.runAppendEntries(i, responeses)
    }

    // collect all responses as they come in
    for {
        select {
        case val := <-responses:
            // on success, we update the match index to the point we found a match
            // then reset nextIndex so the next heartbeat will match farther
            if val.Success {
                n.MatchIndex[val.Id] = n.NextIndex[val.Id] - 1
                n.NextIndex[val.Id] = len(n.Log)
            }
            // we have been demoted, exit and become follower
            else if val.Reason == raft.ErrorCode_AE_OLDTERM {
                n.State = "follower"
                n.Term = val.Term
                n.resetTimer("AE term mismatch")
                return
            }
            // this node ID is out of date, attempt to find earliest matching point
            else if val.Reason == raft.ErrorCode_AE_LOGMISMATCH {
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

func (n *node) runAppendEntries(node_id int, resp chan raft.AppendEntriesResponse) {

    ctx, cancel := context.WithTimeout(context.Background(), n.HeartbeatTimeout*time.Millisecond)
    defer cancel()

    var entries []raft.Entry
    if n.NextIndex[i] == len(n.Log) {
        entries = make([]raft.Entry, 0)
    } else {
        entries = n.Log[n.NextIndex[i]-1:]
    }

    args := raft.AppendEntriesRequest{
        Term: n.Term,
        LeaderId: n.ID,
        PrevLogIndex: n.NextIndex[node_id] - 1
        PrevLogTerm: n.Log[n.NextIndex[node_id] - 1].Term,
        LeaderCommit: n.CommitIndex,
        Entries: entries,
    }

    r, err := n.ServersRaftClient[node_id].AppendEntries(ctx, args)
    if err != nil {
        //TODO HANDLE error
    }

    resp <- r
}

func (n *node) initializeLeader(){
    n.State = "leader"
    n.NextIndex = make([]int32, len(n.ServersAddr))
    n.MatchIndex = make([]int32, len(n.ServersAddr))
    for i, node := range n.ServersAddr {
        n.NextIndex[i] = len(n.Log)
        n.MatchIndex[i] = 0
    }
}

func (n *node) loop() {
    done := make(chan string, 1) // keep track of timeout and success
    n.timer := time.NewTimer(timeout * time.Millisecond)

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
        }
        // candidates elect themselves, on win become a leader and heartbeat
        else if n.State == "candidate" {
            success := n.beginElection(done)
            if success {
                n.initializeLeader()
                n.heartbeat(done)
            }
        } else n.State == "leader" {
            n.heartbeat(done)
        }
    }
}
