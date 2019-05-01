package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	rf "github.com/kpister/raft/raft"
)

func (n *node) persistLog() {
	f, err := os.OpenFile(n.Logfile, os.O_WRONLY|os.O_CREATE, 0644)
	check(err)
	defer f.Close()
	// naive method just rewrite everything once again
	_, err = f.WriteString(strconv.Itoa(int(n.CurrentTerm)) + "\n")
	check(err)

	_, err = f.WriteString(strconv.Itoa(int(n.VotedFor)) + "\n")
	check(err)

	_, err = f.WriteString(strconv.Itoa(len(n.Log)-1) + "\n")
	check(err)

	// write one log entry on each line
	for i := 1; i < len(n.Log); i++ { // don't write the first dummy entry
		f.WriteString(n.Log[i].Command + ":" + strconv.Itoa(int(n.Log[i].Term)) + "\n")
	}

}

func (n *node) isFirstBoot() bool {
	if _, err := os.Stat(n.Logfile); err == nil {
		return false
	}
	return true
}

func (n *node) readCurrentTerm() int32 {
	f, err := os.Open(n.Logfile)
	check(err)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan()
	CurrentTerm := scanner.Text()
	val, _ := strconv.Atoi(CurrentTerm)
	return int32(val)
}

func (n *node) readVotedFor() int32 {

	f, err := os.Open(n.Logfile)
	check(err)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // read first line - term
	scanner.Scan() // read second line - voted for

	VotedFor := scanner.Text()
	val, _ := strconv.Atoi(VotedFor)
	return int32(val)
}

func (n *node) readLog() []*rf.Entry {

	f, err := os.Open(n.Logfile)
	check(err)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // read first line - term
	scanner.Scan() // read second line - voted for
	scanner.Scan() // read third line - num log entries

	numEntries, _ := strconv.Atoi(scanner.Text())

	var logs []*rf.Entry
	// apped a diummy entry to the log
	dummyEntry := rf.Entry{Term: 0, Index: 0, Command: "DUMMY$DUMMY"}
	logs = append(logs, &dummyEntry)

	for i := 0; i < numEntries; i++ {
		scanner.Scan()
		logEntry := scanner.Text() // of form command:term
		logEntrySeperated := strings.Split(logEntry, ":")
		term, _ := strconv.Atoi(logEntrySeperated[1])
		entry := rf.Entry{Term: int32(term), Index: 0, Command: logEntrySeperated[0]}
		logs = append(logs, &entry)
	}
	return logs
}
