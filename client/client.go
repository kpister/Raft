package client

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	kv "github.com/kpister/raft/kvstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// Client is the raft kvstore client
type Client struct {
	kvClient  []kv.KeyValueStoreClient
	conn      *grpc.ClientConn
	clientID  string
	seqNo     int32
	servAddr  string
	timeLogFp *os.File
}

// NewClient creates a new client object
func NewClient() *Client {
	return &Client{
		clientID:  "client",
		seqNo:     rand.Int31n(1000000000),
		servAddr:  "",
		conn:      nil,
		kvClient:  make([]kv.KeyValueStoreClient, 0),
		timeLogFp: nil,
	}
}

// MessagePut sends PUT requests to a server node
func (cl *Client) MessagePut(key string, value string) kv.ReturnCode {
	return cl._MessagePut(key, value, 0, -1)
}

func (cl *Client) _MessagePut(key string, value string, grpcConnID int, seqNumber int32) kv.ReturnCode {
	if seqNumber == -1 {
		seqNumber = cl.seqNo
	}

	task := cl.servAddr + "\tPUT"

	log.Printf("%s request:%s %s\n", task, key, value)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// build request
	req := &kv.PutRequest{
		Key:      key,
		Value:    value,
		From:     -1,
		ClientId: cl.clientID,
		SeqNo:    seqNumber,
	}

	t := time.Now()

	res, err := cl.kvClient[grpcConnID].Put(ctx, req)
	if err != nil {
		log.Printf("%s grpc failed:%v\n", task, err)
		return kv.ReturnCode_FAILURE
	}

	switch res.Ret {
	case kv.ReturnCode_SUCCESS:
		// ONLY Write to output if it succeeded
		log.Printf("%s succeeded\n", task)
		cl.timeTrack(t, task)
		return res.Ret
	case kv.ReturnCode_FAILURE_DEMOTED:
		log.Printf("%s failed: not leader, true leader: %s\n", task, res.LeaderHint)
		// set most recent leader to the leader hint
		log.Printf("Leader changed from to %s", res.LeaderHint)
		mostRecentLeaderAddr = res.LeaderHint
		return res.Ret
	case kv.ReturnCode_FAILURE_EXPIRED:
		log.Printf("%s failed: AE expired\n", task)
		return res.Ret
	case kv.ReturnCode_SUCCESS_SEQNO:
		// ONLY Write to output if it succeeded
		log.Printf("%s succeeded: duplicated seqNo\n", task)
		cl.timeTrack(t, task)
		return res.Ret
	default:
		return kv.ReturnCode_FAILURE
	}
}

// MessageGet sends GET request to the server node
func (cl *Client) MessageGet(key string) (string, kv.ReturnCode) {
	return cl._MessageGet(key, 0)
}

func (cl *Client) _MessageGet(key string, grpcConnIdx int) (string, kv.ReturnCode) {
	task := cl.servAddr + "\tGET"
	defer cl.timeTrack(time.Now(), task)
	log.Printf("%s request:%s\n", task, key)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// build request
	req := &kv.GetRequest{
		Key: key,
	}

	res, err := cl.kvClient[grpcConnIdx].Get(ctx, req, grpc_retry.WithMax(5), grpc_retry.WithPerRetryTimeout(500*time.Millisecond))
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
	elapsed := int64(time.Since(start) / time.Millisecond)
	log.Printf("%s duration:%d\n", task, elapsed)
	if cl.timeLogFp != nil {
		cl.timeLogFp.WriteString(fmt.Sprintf("%s duration:%d\n", task, elapsed))
	}
}

func (cl *Client) _Connect(servAddr string) *grpc.ClientConn {
	if cl.conn != nil {
		cl.conn.Close()
	}

	cl.servAddr = servAddr
	task := cl.servAddr + "\tCONN"
	// grpc will retry in 15 ms at most 5 times when failed
	// TODO: put parameters into config
	opts := []grpc_retry.CallOption{
		grpc_retry.WithMax(5),
		grpc_retry.WithPerRetryTimeout(500 * time.Millisecond),
		grpc_retry.WithCodes(codes.DeadlineExceeded, codes.Unavailable, codes.Canceled),
	}
	conn, err := grpc.Dial(cl.servAddr, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(opts...)))
	if err != nil {
		log.Printf("%s failed:%s\n", task, cl.servAddr)
	}
	log.Printf("%s Connected:%s\n", task, cl.servAddr)

	return conn
}

// Connect connects to the server with the specified server address, and creates a single grpc connection
func (cl *Client) Connect(servAddr string) {
	cl.conn = cl._Connect(servAddr)
	cl.kvClient = make([]kv.KeyValueStoreClient, 0)
	cl.kvClient = append(cl.kvClient, kv.NewKeyValueStoreClient(cl.conn))
}

// ConnectN connects to the server with the specified server address, and creates N grpc connections
// for concurrent grpc requests
func (cl *Client) ConnectN(NumGrpcConn int) {
	cl.conn = cl._Connect(mostRecentLeaderAddr)
	for i := 0; i < NumGrpcConn; i++ {
		cl.kvClient = append(cl.kvClient, kv.NewKeyValueStoreClient(cl.conn))
	}
}

// Close closes the connection
func (cl *Client) Close() {
	cl.conn.Close()
}

// GetSeqNo gets clients's current sequence number
func (cl *Client) GetSeqNo() (int32, string) {
	return cl.seqNo, cl.clientID
}

// SetTimeLog sets the log file for timeTrack
func (cl *Client) SetTimeLog(timeLogFp *os.File) {
	cl.timeLogFp = timeLogFp
}

// runGetRequest issues GET request, and puts response into resps channel no matter it's success or failed
// arg command is "key"
func (cl *Client) runGetRequest(command string, grpcConnIdx int, resps chan bool) {
	cl._MessageGet(command, grpcConnIdx)
	resps <- true
}

// runPutRequest issues PUT request, and put repsonse into resps channel
// it will retry at most 5 times on failures
// arg command is "key$val"
func (cl *Client) runPutRequest(command string, grpcConnIdx int, resps chan bool, seqNumber int32) {
	splits := strings.Split(command, "$")
	key, val := splits[0], splits[1]
	for i := 0; i < 5; i++ {
		ret := cl._MessagePut(key, val, grpcConnIdx, seqNumber)
		if ret == kv.ReturnCode_SUCCESS || ret == kv.ReturnCode_SUCCESS_SEQNO {
			resps <- true
			return
		}
	}
}

var mostRecentLeaderAddr string

// Benchmark measures GET and PUT operation performance
// args command: "PUT key$val" or "GET key"
func Benchmark(command string, leaderAddr string, nClients int, maxConns int, duration time.Duration, timeLogFp *os.File) int {

	// set most recent leader to what is passed in
	mostRecentLeaderAddr = leaderAddr

	var conns []*Client
	defer closeAllConns(conns)

	// command is "PUT key$val" or "GET key"
	splits := strings.SplitN(command, " ", 2)
	op, command := splits[0], splits[1]

	resps := make(chan bool, nClients*maxConns)
	timer := time.NewTimer(duration)
	seqNumber := (int32)(rand.Int31n(1000000000))
	for i := 0; i < nClients; i++ {
		c := NewClient()
		conns = append(conns, c)

		c.SetTimeLog(timeLogFp)
		c.SetClientID("benchmark" + strconv.Itoa(i))
		// defer c.Close()

		c.ConnectN(maxConns)
		for j := 0; j < maxConns; j++ {
			if op == "GET" {
				go c.runGetRequest(command, j, resps)
			} else {
				go c.runPutRequest(command, j, resps, seqNumber)
				seqNumber++
			}
		}
	}

	log.Println("FINISHED ISSUING ALL THE REQURESTS, WAITING FOR REPLES")

	nResps := 0
	for {
		select {
		case <-timer.C:
			return nResps
		case <-resps:
			nResps++
		}
	}
}

func closeAllConns(conns []*Client) {
	for _, con := range conns {
		con.Close()
	}
}
