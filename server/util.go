package main

import (
	"math/rand"

	"github.com/kpister/raft/raft"
)

/*
Returns true if the message should be dropped otherwise returns false
*/
func (n *node) dropMessageChaos(from int32) bool {
	if from == -1 {
		return false
	}
	random0to1 := rand.Float32()
	// n.Chaos is the probability with which we want to drop the value
	// 0 - no drop 1 - drop every message
	if random0to1 > n.Chaos[from][n.ID] {
		return false
	}
	return true
}

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func isEqual(e1 *raft.Entry, e2 *raft.Entry) bool {
	if e1.Term == e2.Term {
		return true
	}
	return false
}

func resizeSlice(a []*raft.Entry, newSize int) []*raft.Entry {
	return append([]*raft.Entry(nil), a[:newSize]...)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
