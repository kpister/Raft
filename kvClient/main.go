package main

import (
	"bufio"
	"fmt"
	"github.com/kpister/raft/client"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

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
	client := client.NewClient()
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

			client.Connect(splits[1])

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
