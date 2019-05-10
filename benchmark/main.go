package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kpister/raft/client"
	kv "github.com/kpister/raft/kvstore"
)

type config struct {
	ServersAddr []string
	NumClients  int
	NumConns    int
	Duration    int
	KeySize     int
	ValSize     int
	NumEntries  int
	operation   string
}

var benchmarkFile string

func _findLeader(serversAddr []string) (int, int) {
	c := client.NewClient()
	c.SetClientID("findLeader")
	cntLeader := 0
	leaderIndex := -1
	for i, servAddr := range serversAddr {
		c.Connect(servAddr)
		ret := c.MessagePut("find"+strconv.Itoa((int)(rand.Int31n(100000))), "find"+strconv.Itoa(i))
		if ret == kv.ReturnCode_SUCCESS || ret == kv.ReturnCode_SUCCESS_SEQNO {
			cntLeader++
			leaderIndex = i
		}
	}
	return leaderIndex, cntLeader
}

func findLeader(serversAddr []string) (int, bool) {
	for i := 0; i < 5; i++ {
		leaderid, numleaders := _findLeader(serversAddr)
		if numleaders == 1 {
			return leaderid, true
		}
	}
	return -1, false
}

func init() {
	flag.StringVar(&benchmarkFile, "config", "benchmark.json", "benchmark configuration")
}

// RunBenchmark runs measurement by benchmarkFile
func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	var conf config

	flag.Parse()

	benchmarkFile, _ = filepath.Abs(benchmarkFile)
	configData, err := ioutil.ReadFile(benchmarkFile)
	if err != nil {
		log.Fatal("Open config file error.")
	}
	err = json.Unmarshal(configData, &conf)
	if err != nil {
		log.Fatal("Unmarshal config file error.")
	}

	leaderID, leadFound := findLeader(conf.ServersAddr)
	if !leadFound {
		log.Fatal("Failed to find leader.")
	}

	log.Printf("Leader: %s\n", conf.ServersAddr[leaderID])

	for i := 0; i < conf.NumEntries; i++ {
		// command "PUT key$val" or "GET key"
		key := fmt.Sprintf("%0"+strconv.Itoa(conf.KeySize)+"d", i)
		val := fmt.Sprintf("%0"+strconv.Itoa(conf.ValSize)+"d", i)
		command := conf.operation + " " + key + "$" + val
		client.Benchmark(command, conf.ServersAddr[leaderID], conf.NumClients, conf.NumConns, time.Duration(conf.Duration)*time.Second)
	}
}
