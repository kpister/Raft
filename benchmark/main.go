package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
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
	Operation   string
}

var conf config
var benchmarkFile string
var timeLogDir string

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
	rand.Seed(time.Now().UTC().UnixNano())

	flag.StringVar(&benchmarkFile, "config", "benchmark.json", "benchmark configuration")
	flag.StringVar(&timeLogDir, "time_log_dir", "time_log/", "benchmark log director")
	flag.Parse()
}

// RunBenchmark runs measurement by benchmarkFile
func main() {
	benchmarkFile, _ = filepath.Abs(benchmarkFile)
	configData, err := ioutil.ReadFile(benchmarkFile)
	if err != nil {
		log.Fatal("Open config file error.")
	}
	err = json.Unmarshal(configData, &conf)
	if err != nil {
		log.Fatal("Unmarshal config file error.")
	}

	// log setup
	os.Mkdir(timeLogDir, 0700)
	timeLogFp, err := os.OpenFile(timeLogDir+conf.Operation+"_"+time.Now().Format("2006.01.02_15:04:05.time"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Open log file error: %v\n", err)
	}

	for i := 0; i < conf.NumEntries; i++ {
		// command "PUT key$val" or "GET key"

		leaderID, leadFound := findLeader(conf.ServersAddr)
		if !leadFound {
			log.Fatal("Failed to find leader.")
		}

		log.Printf("Leader: %s\n", conf.ServersAddr[leaderID])

		command := conf.Operation + " "
		if conf.Operation == "PUT" {
			key := fmt.Sprintf("%0"+strconv.Itoa(conf.KeySize)+"d", i)
			val := fmt.Sprintf("%0"+strconv.Itoa(conf.ValSize)+"d", i)
			command += key + "$" + val
		} else {
			key := fmt.Sprintf("%0"+strconv.Itoa(conf.KeySize)+"d", i)
			command += key
		}
		fmt.Println(command)
		client.Benchmark(command, conf.ServersAddr[leaderID], conf.NumClients, conf.NumConns, time.Duration(conf.Duration)*time.Second, timeLogFp)
	}
}
