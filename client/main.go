package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	kv "github.com/kpister/raft/kvstore"
	"google.golang.org/grpc"
)

func messagePut(c kv.KeyValueStoreClient, key string, value string) {
	task := servAddr + "\tPUT"
	defer timeTrack(time.Now(), task)
	log.Printf("%s request:%s %s\n", task, key, value)
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
		log.Printf("%s grpc failed:%v\n", task, err)
		return
	}

	if res.Ret == kv.ReturnCode_SUCCESS {
		log.Printf("%s respond\n", task)
	} else {
		log.Printf("%s failed\n", task)
	}
}

func messageGet(c kv.KeyValueStoreClient, key string) string {
	task := servAddr + "\tGET"
	defer timeTrack(time.Now(), task)
	log.Printf("%s request:%s\n", task, key)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// build request
	req := &kv.GetRequest{
		Key: key,
	}

	res, err := c.Get(ctx, req)
	if err != nil {
		log.Printf("%s grpc failed:%v\n", task, err)
		return ""
	}

	if res.Ret == kv.ReturnCode_SUCCESS {
		log.Printf("%s respond:%s\n", task, res.Value)
	} else {
		log.Printf("%s failed\n", task)
	}
	return res.Value
}

func timeTrack(start time.Time, task string) {
	elapsed := time.Since(start)
	log.Printf("%s duration:%s", task, elapsed)
}

func connect() (*grpc.ClientConn, kv.KeyValueStoreClient) {
	task := servAddr + "\tCONN"
	conn, err := grpc.Dial(servAddr, grpc.WithInsecure())
	if err != nil {
		log.Printf("%s failed:%s\n", task, servAddr)
	}

	log.Printf("%s connected:%s\n", task, servAddr)
	kvClient := kv.NewKeyValueStoreClient(conn)
	return conn, kvClient
}

var servAddr string

// Client commands:
// 1. CONN host:port
// 2. PUT key val
// 3. GET key
func main() {
	// log setup
	f, err := os.OpenFile("log/"+time.Now().Format("2006.01.02_15:04:05.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Open log file error: %v\n", err)
	}
	defer f.Close()
	mw := io.MultiWriter(os.Stderr, f)
	log.SetOutput(mw)

	inputReader := bufio.NewReader(os.Stdin)
	var conn *grpc.ClientConn
	var kvClient kv.KeyValueStoreClient
	for {
		in, _ := inputReader.ReadString('\n')
		in = strings.TrimSpace(in)

		splits := strings.Split(in, " ")
		if len(splits) <= 1 {
			fmt.Println("bad input")
			continue
		}

		switch splits[0] {
		// CONN host:port
		case "CONN":
			if len(splits) != 2 {
				fmt.Println("bad input")
				continue
			}

			servAddr = splits[1]
			if conn != nil {
				conn.Close()
			}
			conn, kvClient = connect()
			defer conn.Close()
			break

		// PUT key val
		case "PUT":
			if len(splits) != 3 {
				fmt.Println("bad input")
				continue
			}

			key, val := splits[1], splits[2]
			messagePut(kvClient, key, val)
			break

		// GET key
		case "GET":
			if len(splits) != 2 {
				fmt.Println("bad input")
				continue
			}

			key := splits[1]
			messageGet(kvClient, key)
			break
		}
	}
}
