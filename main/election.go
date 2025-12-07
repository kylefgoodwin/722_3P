package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/google/uuid"
)

// --- CONFIGURATION ---
const (
	ZKHosts            = "localhost:2181"
	ElectionPath       = "/election"
	DeathTimestampFile = "last_leader_death.txt"
	ColdStartFile      = "cold_start_data.csv"
	FailoverFile       = "failover_data.csv"
	LeaderTenure       = 5 * time.Second
)

func main() {
	runNo := flag.Int("runNo", 1, "Test Run Number")
	flag.Parse()

	var leaderDied = false
	var leaderDeathTime time.Time

	// 1. Connect to Zookeeper
	conn, _, err := zk.Connect([]string{ZKHosts}, time.Second*5)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// 3. Volunteer (Create my ephemeral node)
	myGUID := uuid.New().String()
	prefix := fmt.Sprintf("%s/guid-n_", ElectionPath)
	myPath, err := conn.Create(prefix, []byte{}, zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	if err != nil {
		panic(err)
	}

	fmt.Printf("[%s] My Path: %s\n", myGUID[:8], myPath)

	// State variables
	startTime := time.Now()
	coldStartLogged := false

	// --- MAIN LOOP ---
	for {
		// A. Get all nodes in election
		children, _, err := conn.Children(ElectionPath)
		if err != nil || len(children) == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// B. Sort them to find the leader
		sort.Slice(children, func(i, j int) bool {
			return extractSeq(children[i]) < extractSeq(children[j])
		})

		currLeader := children[0]

		// C. Find where I am in the line
		myIndex := -1
		myNodeName := strings.Split(myPath, "/")[2] // e.g. "guid-n_000000001"

		for i, child := range children {
			if child == myNodeName {
				myIndex = i
				break
			}
		}

		if !coldStartLogged {
			discoverTime := time.Since(startTime).Nanoseconds()
			logToCSV(ColdStartFile, strconv.Itoa(*runNo), myGUID[:8], currLeader, float64(discoverTime)/1e6)
			coldStartLogged = true
		}

		if leaderDied {
			fmt.Printf("%.2f", float64(time.Now().UnixNano()))
			// failoverTime := time.Since(leaderDeathTime).Nanoseconds()
			failoverTime := time.Now().UnixNano() - leaderDeathTime.UnixNano()
			logToCSV(FailoverFile, strconv.Itoa(*runNo), myGUID[:8], currLeader, float64(failoverTime)/1e6)
			fmt.Printf("[%s] Failover complete! New leader: %s (took %.2f ms)\n",
				myGUID[:8], currLeader, float64(failoverTime)/1e6)
			os.Exit(0)
		}

		// D. Election Logic
		if myIndex == 0 {
			fmt.Printf("\n[%s] === I AM THE LEADER ===\n", myGUID[:8])

			// Stay leader for 5 seconds then crash
			time.Sleep(LeaderTenure)

			// Simulate Crash: Write time of death and exit
			writeDeathTimestamp()
			fmt.Printf("[%s] Simulating CRASH at %v...\n", myGUID[:8], time.Now())
			conn.Close()
			os.Exit(0)

		} else {
			// WATCH the current leader (not node ahead of me)
			watchPath := ElectionPath + "/" + currLeader

			fmt.Printf("Follower [%s], watching leader %s\n", myGUID[:8], watchPath)

			exists, _, eventChan, err := conn.ExistsW(watchPath)
			if err != nil {
				panic(err)
			}

			if exists {
				// Block until leader is deleted
				event := <-eventChan
				if event.Type == zk.EventNodeDeleted {
					// Mark that leader died and read the death timestamp
					fmt.Printf("[%s] Leader died! Checking for new leader...\n", myGUID[:8])
					leaderDeathTime = readDeathTimestamp()
					// fmt.Printf("[%s] Recorded leader death time: %v\n", myGUID[:8], leaderDeathTime)
					leaderDied = true
					continue
				}
			}
		}
	}
}

// --- HELPER FUNCTIONS ---

func logToCSV(filename, run, nodeID, leaderNode string, duration float64) {
	// Simple append to file
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	// Write header only if file is empty
	stat, _ := f.Stat()
	if stat.Size() == 0 {
		f.WriteString("Run No,Node ID,Leader Node,Duration (ms)\n")
	}

	f.WriteString(fmt.Sprintf("%s,%s,%s,%.5f\n", run, nodeID, leaderNode, duration))
}

func writeDeathTimestamp() {
	now := time.Now().UnixNano()
	os.WriteFile(DeathTimestampFile, []byte(fmt.Sprintf("%d", now)), 0644)
}

func readDeathTimestamp() time.Time {
	data, err := os.ReadFile(DeathTimestampFile)
	if err != nil {
		return time.Now()
	}
	var timestamp int64
	fmt.Sscanf(string(data), "%d", &timestamp)
	return time.Unix(0, timestamp)
}

func extractSeq(nodeName string) int {
	parts := strings.Split(nodeName, "_")
	if len(parts) < 2 {
		return 0
	}
	seq, _ := strconv.Atoi(parts[len(parts)-1])
	return seq
}
