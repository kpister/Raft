package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	kv "github.com/kpister/raft/kvstore"
	"google.golang.org/grpc"
)

// Client is the raft kvstore client
type Client struct {
	kvClient kv.KeyValueStoreClient
	conn     *grpc.ClientConn
	clientID string
	seqNo    int32
	servAddr string
}

// NewClient creates a new client object
func NewClient() *Client {
	return &Client{
		clientID: "client",
		seqNo:    0,
		servAddr: "",
	}
}

// MessagePut sends PUT requests to a server node
func (cl *Client) MessagePut(key string, value string) bool {
	task := cl.servAddr + "\tPUT"
	defer cl.timeTrack(time.Now(), task)
	log.Printf("%s request:%s %s\n", task, key, value)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// build request
	req := &kv.PutRequest{
		Key:      key,
		Value:    value,
		From:     -1,
		ClientId: cl.clientID,
		SeqNo:    cl.seqNo,
	}

	res, err := cl.kvClient.Put(ctx, req)
	if err != nil {
		log.Printf("%s grpc failed:%v\n", task, err)
		return false
	}

	switch res.Ret {
	case kv.ReturnCode_SUCCESS:
		log.Printf("%s succeeded\n", task)
		return true
	case kv.ReturnCode_FAILURE_DEMOTED:
		log.Printf("%s failed: not leader, true leader: %s\n", task, res.LeaderHint)
		return false
	case kv.ReturnCode_FAILURE_EXPIRED:
		log.Printf("%s failed: AE expired\n", task)
		return false
	case kv.ReturnCode_SUCCESS_SEQNO:
		log.Printf("%s succeeded: duplicated seqNo\n", task)
		return true
	default:
		return false
	}
}

// MessageGet sends GET request to the server node
func (cl *Client) MessageGet(key string) (string, bool) {
	task := cl.servAddr + "\tGET"
	defer cl.timeTrack(time.Now(), task)
	log.Printf("%s request:%s\n", task, key)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// build request
	req := &kv.GetRequest{
		Key: key,
	}

	res, err := cl.kvClient.Get(ctx, req)
	if err != nil {
		log.Printf("%s grpc failed:%v\n", task, err)
		return "", false
	}

	if res.Ret == kv.ReturnCode_SUCCESS {
		log.Printf("%s respond:%s\n", task, res.Value)
		return res.Value, true
	}

	log.Printf("%s failed\n", task)
	return "", false
}

// IncrementSeqNo increments the sequence number by 1
func (cl *Client) IncrementSeqNo() {
	cl.seqNo++
}

// SetClientID sets the client identifier
func (cl *Client) SetClientID(id string) {
	cl.clientID = id
}

func (cl *Client) timeTrack(start time.Time, task string) {
	elapsed := time.Since(start)
	log.Printf("%s duration:%s", task, elapsed)
}

// Connect connects to the server with the specified server address
func (cl *Client) Connect(servAddr string) {
	cl.servAddr = servAddr
	task := cl.servAddr + "\tCONN"
	// grpc will retry in 15 ms at most 5 times when failed
	// TODO: put parameters into config
	opts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffLinear(time.Duration(15 * time.Millisecond))),
		grpc_retry.WithMax(5),
	}
	conn, err := grpc.Dial(cl.servAddr, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(opts...)))
	if err != nil {
		log.Printf("%s failed:%s\n", task, cl.servAddr)
	}

	log.Printf("%s Connected:%s\n", task, cl.servAddr)
	kvClient := kv.NewKeyValueStoreClient(conn)

	cl.conn = conn
	cl.kvClient = kvClient
}

// Client commands:
// 1. CONN host:port
// 2. PUT key val
// 3. GET key
// 4. ID id
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
	client := NewClient()
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

			if client.conn != nil {
				client.conn.Close()
			}
			client.Connect(splits[1])
			defer client.conn.Close()

		// PUT key val
		case "PUT":
			if len(splits) != 3 {
				fmt.Println("bad input")
				continue
			}

			key, val := splits[1], splits[2]
			client.MessagePut(key, val)
			client.IncrementSeqNo()

		// GET key
		case "GET":
			if len(splits) != 2 {
				fmt.Println("bad input")
				continue
			}

			key := splits[1]
			client.MessageGet(key)

		// set client ID
		// ID clientID
		case "ID":
			if len(splits) != 2 {
				fmt.Println("bad input")
				continue
			}

			client.SetClientID(splits[1])
		}
	}
}
