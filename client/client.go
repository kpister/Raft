package client

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	kv "github.com/kpister/raft/kvstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
		seqNo:    rand.Int31n(1000),
		servAddr: "",
		conn:     nil,
	}
}

// MessagePut sends PUT requests to a server node
func (cl *Client) MessagePut(key string, value string) kv.ReturnCode {
	task := cl.servAddr + "\tPUT"
	defer cl.timeTrack(time.Now(), task)
	log.Printf("%s request:%s %s\n", task, key, value)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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
		return kv.ReturnCode_FAILURE
	}

	switch res.Ret {
	case kv.ReturnCode_SUCCESS:
		log.Printf("%s succeeded\n", task)
		return res.Ret
	case kv.ReturnCode_FAILURE_DEMOTED:
		log.Printf("%s failed: not leader, true leader: %s\n", task, res.LeaderHint)
		return res.Ret
	case kv.ReturnCode_FAILURE_EXPIRED:
		log.Printf("%s failed: AE expired\n", task)
		return res.Ret
	case kv.ReturnCode_SUCCESS_SEQNO:
		log.Printf("%s succeeded: duplicated seqNo\n", task)
		return res.Ret
	default:
		return kv.ReturnCode_FAILURE
	}
}

// MessageGet sends GET request to the server node
func (cl *Client) MessageGet(key string) (string, kv.ReturnCode) {
	task := cl.servAddr + "\tGET"
	defer cl.timeTrack(time.Now(), task)
	log.Printf("%s request:%s\n", task, key)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// build request
	req := &kv.GetRequest{
		Key: key,
	}

	res, err := cl.kvClient.Get(ctx, req, grpc_retry.WithMax(5), grpc_retry.WithPerRetryTimeout(500*time.Millisecond))
	if err != nil {
		log.Printf("%s grpc failed:%v\n", task, err)
		return "", kv.ReturnCode_FAILURE
	}

	if res.Ret == kv.ReturnCode_SUCCESS {
		log.Printf("%s respond:%s\n", task, res.Value)
		return res.Value, kv.ReturnCode_SUCCESS
	}

	log.Printf("%s failed\n", task)
	return "", kv.ReturnCode_FAILURE
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
	if cl.conn != nil {
		cl.conn.Close()
	}

	cl.servAddr = servAddr
	task := cl.servAddr + "\tCONN"
	// grpc will retry in 15 ms at most 5 times when failed
	// TODO: put parameters into config
	opts := []grpc_retry.CallOption{
		grpc_retry.WithMax(5),
		grpc_retry.WithPerRetryTimeout(150 * time.Millisecond),
		grpc_retry.WithCodes(codes.DeadlineExceeded, codes.Unavailable, codes.Canceled),
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

// Close closes the connection
func (cl *Client) Close() {
	cl.conn.Close()
}
