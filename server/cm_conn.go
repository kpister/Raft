package main

import (
	"context"
	"log"

	cm "github.com/kpister/raft/chaosmonkey"
)

func (n *node) UploadMatrix(ctx context.Context, mat *cm.ConnMatrix) (*cm.Status, error) {
	if len(mat.Rows) != len(n.ServersAddr) {
		return &cm.Status{Ret: cm.StatusCode_ERROR}, nil
	}

	for i := 0; i < len(mat.Rows); i++ {
		for j := 0; j < len(mat.Rows[i].Vals); j++ {
			if len(mat.Rows[i].Vals) != len(n.ServersAddr) {
				return &cm.Status{Ret: cm.StatusCode_ERROR}, nil
			}
			n.Chaos[i][j] = mat.Rows[i].Vals[j]
		}
	}
	log.Printf("UL_MAT\n")

	return &cm.Status{Ret: cm.StatusCode_OK}, nil
}

func (n *node) UpdateValue(ctx context.Context, matv *cm.MatValue) (*cm.Status, error) {
	if matv.Row < 0 || int(matv.Row) >= len(n.ServersAddr) || matv.Col < 0 || int(matv.Col) >= len(n.ServersAddr) {
		return &cm.Status{Ret: cm.StatusCode_ERROR}, nil
	}

	n.Chaos[matv.Row][matv.Col] = matv.Val
	log.Printf("UD_MAT:%d, %d, %f\n", matv.Row, matv.Col, matv.Val)

	return &cm.Status{Ret: cm.StatusCode_OK}, nil
}

func (n *node) GetState(ctx context.Context, in *cm.EmptyMessage) (*cm.ServerState, error) {
	response := &cm.ServerState{}
	response.ID = n.ID
	response.State = n.State
	response.CurrentTerm = n.CurrentTerm
	response.VotedFor = n.VotedFor
	response.LeaderID = n.LeaderID
	response.CommitIndex = n.CommitIndex
	response.LastApplied = n.LastApplied
	response.MatchIndex = n.MatchIndex
	response.NextIndex = n.NextIndex
	response.Log = n.Log

	return response, nil
}
