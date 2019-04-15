package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	cm "github.com/kpister/raft/chaosmonkey"
	kv "github.com/kpister/raft/kvstore"
	"google.golang.org/grpc"
)

func messagePut(c kv.KeyValueStoreClient, key string, value string) {
	fmt.Printf("Putting value: %s\n", value)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// build request
	req := &kv.PutRequest{
		Key:   key,
		Value: value,
		From:  -1,
	}

	res, err := c.Put(ctx, req)
	if err != nil {
		log.Fatalf("Put operation failed: %v\n", err)
	}
	fmt.Printf("Status code: %d\n", res.Ret)
}

func messageGet(c kv.KeyValueStoreClient, key string) string {
	fmt.Printf("Getting value for key %s\n", key)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// build request
	req := &kv.GetRequest{
		Key: key,
	}

	res, err := c.Get(ctx, req)
	if err != nil {
		log.Fatalf("Get operation failed: %v\n", err)
	}
	fmt.Printf("Status code: %d\n", res.Ret)
	fmt.Printf("Value: %s\n", res.Value)
	return res.Value
}

func uploadMatrix(c cm.ChaosMonkeyClient, matPath string) {
	// read matrix from file
	matFp, err := os.Open(matPath)
	if err != nil {
		log.Fatalf("Open matrix file error: %v\n", err)
	}
	defer matFp.Close()

	scanner := bufio.NewScanner(matFp)
	mat := &cm.ConnMatrix{
		From: -1,
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		sps := strings.Split(line, " ")
		row := cm.ConnMatrix_MatRow{}
		for _, sp := range sps {
			p, _ := strconv.ParseFloat(sp, 32)
			row.Vals = append(row.Vals, float32(p))
		}
		mat.Rows = append(mat.Rows, &row)
	}

	fmt.Printf("Uploading connectivity matrix %s\n", matPath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := c.UploadMatrix(ctx, mat)
	if err != nil {
		log.Fatalf("Upload matrix operation failed: %v\n", err)
	}
	fmt.Printf("Status code: %d\n", res.Ret)
}

func updateValue(c cm.ChaosMonkeyClient, row int32, col int32, val float32) {
	matv := &cm.MatValue{
		Row:  row,
		Col:  col,
		Val:  val,
		From: -1,
	}

	fmt.Printf("Updating connectivity matrix at (%d, %d) with %f\n", row, col, val)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := c.UpdateValue(ctx, matv)
	if err != nil {
		log.Fatalf("Update value operation failed: %v\n", err)
	}
	fmt.Printf("Status code: %d\n", res.Ret)
}

const (
	servAddr = "localhost:8801"
)

func main() {
	conn, err := grpc.Dial(servAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v\n", err)
	}
	defer conn.Close()

	kvClient := kv.NewKeyValueStoreClient(conn)
	messagePut(kvClient, "m", "1")
	messageGet(kvClient, "m")
	messageGet(kvClient, "n")

	cmClient := cm.NewChaosMonkeyClient(conn)
	uploadMatrix(cmClient, "mat1.txt")
	updateValue(cmClient, 0, 1, 0.5)

	messagePut(kvClient, "n", "2")
}
