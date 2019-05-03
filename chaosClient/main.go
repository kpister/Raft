package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cm "github.com/kpister/raft/chaosmonkey"
	"google.golang.org/grpc"
)

type config struct {
	ServersAddr []string
}

type key struct {
	from int
	to   int
}

func readMatrix(matPath string) *cm.ConnMatrix {

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

	return mat
}

func matToConnMatrix(mat [][]float32) *cm.ConnMatrix {
	conmat := &cm.ConnMatrix{From: -1}

	for i := 0; i < len(mat); i++ {
		for j := 0; j < len(mat[0]); j++ {
			conmat.Rows[i].Vals[j] = mat[i][j]
		}
	}

	return conmat
}

func uploadMatrix(c cm.ChaosMonkeyClient, matPath string) {
	// read matrix from file
	mat := readMatrix(matPath)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Printf("Uploading connectivity matrix %s\n", matPath)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Printf("Updating connectivity matrix at (%d, %d) with %f\n", row, col, val)
	res, err := c.UpdateValue(ctx, matv)
	if err != nil {
		log.Fatalf("Update value operation failed: %v\n", err)
	}
	fmt.Printf("Status code: %d\n", res.Ret)
}

func contains(ids []int, id int) bool {
	for i := 0; i < len(ids); i++ {
		if ids[i] == id {
			return true
		}
	}
	return false
}

func createPartition(numServers int, nodeids []int, mat *cm.ConnMatrix) map[key]float32 {

	previous := make(map[key]float32)

	for i := 0; i < numServers; i++ {
		for j := 0; j < numServers; j++ {
			if (contains(nodeids, i) && !contains(nodeids, j)) || (!contains(nodeids, i) && contains(nodeids, j)) {
				previous[key{from: i, to: j}] = mat.Rows[i].Vals[j]
				mat.Rows[i].Vals[j] = 1.0
			}
		}
	}

	return previous
}

func undoPartition(previous map[key]float32, mat *cm.ConnMatrix) {
	// pass in the previous map returned from the createPartition funciton
	for key, val := range previous {
		mat.Rows[key.from].Vals[key.to] = val
	}
}

func isolateNode(numServers int, nodeid int, mat *cm.ConnMatrix) map[key]float32 {

	previous := make(map[key]float32)

	for i := 0; i < numServers; i++ {
		// to nodeid -- drop
		previous[key{from: i, to: nodeid}] = mat.Rows[i].Vals[nodeid]
		mat.Rows[i].Vals[nodeid] = 1.0
		// from nodeid -- drop
		previous[key{from: nodeid, to: i}] = mat.Rows[nodeid].Vals[i]
		mat.Rows[nodeid].Vals[i] = 1.0
	}

	return previous
}

func deisolateNode(previous map[key]float32, mat *cm.ConnMatrix) {
	// pass in the previous map returned from isolateNode function
	for key, val := range previous {
		mat.Rows[key.from].Vals[key.to] = val
	}
}

func conntectServers(conf config) []cm.ChaosMonkeyClient {

	numServers := len(conf.ServersAddr)
	clients := make([]cm.ChaosMonkeyClient, numServers)

	for i := 0; i < numServers; i++ {
		conn, err := grpc.Dial(conf.ServersAddr[i], grpc.WithInsecure())
		if err != nil {
			log.Fatalf("Failed to connect: %v\n", err)
		}

		clients[i] = cm.NewChaosMonkeyClient(conn)
	}

	return clients
}

// func getState(clients []cm.ChaosMonkeyClient) {
//     for i := 0; i < len(clients); i++ {
//         ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//         defer cancel()

//         res, err := clients[i].GetState(ctx, &cm.EmptyMessage{})
//         if err != nil {
//             log.Fatalf("Get state operation failed on server %v: %v\n", i, err)
//         }
//         fmt.Printf("%v\n", res.ID)
//         fmt.Printf("%v\n", res)
//     }

// }

func sendToServers(clients []cm.ChaosMonkeyClient, mat *cm.ConnMatrix) {
	for i := 0; i < len(clients); i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := clients[i].UploadMatrix(ctx, mat)
		if err != nil {
			log.Fatalf("Upload matrix operation failed: %v\n", err)
		}

		fmt.Printf("Status code: %s\n", res.Ret)
	}
}

func createMatrix(n int, val float32) *cm.ConnMatrix {

	// assert(0.0 <= val && val <= 1.0, "bad value in createMatrix")

	mat := &cm.ConnMatrix{
		From: -1,
	}
	for i := 0; i < n; i++ {
		row := cm.ConnMatrix_MatRow{}
		for j := 0; j < n; j++ {
			row.Vals = append(row.Vals, val)
		}
		mat.Rows = append(mat.Rows, &row)
	}

	return mat
}

func randomizeNetwork(mat *cm.ConnMatrix) {
	for i := 0; i < len(mat.Rows); i++ {
		for j := 0; j < len(mat.Rows); j++ {
			randVal := rand.Float32()
			mat.Rows[i].Vals[j] = randVal
		}
	}
}

func copyMat(src *cm.ConnMatrix, dst *cm.ConnMatrix) {
	for i := 0; i < len(src.Rows); i++ {
		for j := 0; j < len(src.Rows); j++ {
			dst.Rows[i].Vals[j] = src.Rows[i].Vals[j]
		}
	}
}

var configFile string
var chaosInput string

func init() {
	flag.StringVar(&configFile, "config", "chaosConfig.json", "path of the config file")
	flag.StringVar(&chaosInput, "input", "chaosInput", "index of the server")
}

func main() {

	var conf config
	// a map to save matrix configurations
	// matStorage := make(map[string]cm.ConnMatrix)

	flag.Parse()

	configFile, _ = filepath.Abs(configFile)
	chaosInput, _ = filepath.Abs(chaosInput)

	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(configData, &conf)
	if err != nil {
		fmt.Println(err)
	}

	numServers := len(conf.ServersAddr)
	fmt.Println(numServers)

	// where we store the state of the servers
	serverStates := make([]*cm.ServerState, numServers)

	// // initialize the chaos matrix with all zeros
	mat := createMatrix(numServers, 0.0)
	oldMat := createMatrix(numServers, 0.0) // to store the state before partition
	// connect to all servers
	clients := conntectServers(conf)
	// send the default matrix to servers
	sendToServers(clients, mat)

	// randomizeNetwork(mat)

	// sendToServers(clients, mat)
	// time.Sleep(1000)

	// // change the mat as you want
	// part1 := createPartition(numServers, []int{0, 2}, mat)

	// sendToServers(clients, mat)
	// time.Sleep(1000)

	// undoPartition(part1, mat)

	// sendToServers(clients, mat)
	// time.Sleep(1000)

	chaosInputData, err := os.Open(chaosInput)
	if err != nil {
		log.Fatalf("Open config file error: %v\n", err)
	}
	defer chaosInputData.Close()

	scanner := bufio.NewScanner(chaosInputData)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		seperatedCommand := strings.Split(command, " ")
		switch seperatedCommand[0] {
		case "//":
			continue
		case "STATE":
			log.Println("STATE")
			getState(clients, serverStates)
		case "CREATE":
			log.Println("CREATE")
			val, _ := strconv.ParseFloat(seperatedCommand[1], 32)
			mat = createMatrix(numServers, float32(val))
		case "SLEEP":
			log.Println("SLEEP")
			// send the most recent mat to servers
			// sendToServers only happens before a sleep
			sendToServers(clients, mat)
			val, _ := strconv.Atoi(seperatedCommand[1])
			time.Sleep(time.Duration(val) * time.Millisecond)
		case "PARTITION":
			log.Println("PARTITION")
			numPartitions, _ := strconv.Atoi(seperatedCommand[1])
			// partitionName := seperatedCommand[2]
			// save the partition
			copyMat(mat, oldMat)
			// next numPartitions lines would have the paritions
			for i := 0; i < numPartitions; i++ {
				scanner.Scan()
				partition := strings.Split(scanner.Text(), " ")
				ids := make([]int, len(partition))
				for i, id := range partition {
					ids[i], _ = strconv.Atoi(id)
				}
				createPartition(numServers, ids, mat)
			}
		case "HEAL":
			log.Println("HEAL")
			copyMat(oldMat, mat)

		case "RANDOMIZE":
			log.Println("RANDOMIZE")
			randomizeNetwork(mat)
		case "ISOLATE":
			log.Println("ISOLATE")
			val, _ := strconv.Atoi(seperatedCommand[1])
			copyMat(mat, oldMat)
			createPartition(numServers, []int{val}, mat)
		case "ASSERT":
			switch seperatedCommand[1] {
			case "CLIENT_FUNCTIONALITY":
				leaderid, numleaders := findActiveLeader(conf.ServersAddr, clients)
				if numleaders == 1 {
					clientFunctionality("localhost:800" + strconv.Itoa(leaderid))
				} else {
					log.Println("ERROR: CLIENT FUNC TEST FAILED DUE TO NOT 1 leader")
				}

			case "LEADER_FUNCTIONALITY":
				leaderFunctionality(conf.ServersAddr, clients)
			case "LOG_CONSISTENCY":
				leaderid, numleaders := findActiveLeader(conf.ServersAddr, clients)
				if numleaders == 1 {
					logConsistency(serverStates, int32(leaderid))
				} else {
					log.Println("ERROR: LOG CONS TEST FAILED DUE TO NOT 1 leader")
				}
			}

		}
	}

	// cmClient := cm.NewChaosMonkeyClient(conn)
	// uploadMatrix(cmClient, "mat1.txt")
	// updateValue(cmClient, 0, 1, 0.5)

}
