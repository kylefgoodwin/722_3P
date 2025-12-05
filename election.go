package main

import (
	//"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
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
	LeaderTenure       = 15 * time.Second
)

func main() {
	// 1. Connect to Zookeeper
	conn, _, err := zk.Connect([]string{ZKHosts}, time.Second*5)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// 2. Create the Election Parent Node (if it doesn't exist)
	exists, _, _ := conn.Exists(ElectionPath)
	if !exists {
		conn.Create(ElectionPath, []byte{}, 0, zk.WorldACL(zk.PermAll))
	}

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
	failoverLogged := false

	// --- MAIN LOOP ---
	// Instead of complex watchers, we just check the status every 100ms.
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

		// C. Find where I am in the line
		myIndex := -1
		myNodeName := strings.Split(myPath, "/")[2] // e.g. "guid-n_000000001"
		
		for i, child := range children {
			if child == myNodeName {
				myIndex = i
				break
			}
		}

		if myIndex == -1 {
			// I was deleted? Re-volunteer or exit.
			log.Fatal("My node disappeared!")
		}

		// --- DATA COLLECTION 1: COLD START ---
		if !coldStartLogged {
			duration := time.Since(startTime).Seconds()
			if duration < 30 {
				logToCSV(ColdStartFile, "Cold_Start_Time", duration)
				coldStartLogged = true
			}
		}

		// --- DATA COLLECTION 2: FAILOVER ---
		// Check the death file to see if a leader died recently
		content, err := ioutil.ReadFile(DeathTimestampFile)
		if err == nil {
			tsStr := strings.TrimSpace(string(content))
			if deathTimeFloat, err := strconv.ParseFloat(tsStr, 64); err == nil {
				deathTime := unixFloatToTime(deathTimeFloat)
				failoverDuration := time.Since(deathTime).Seconds()

				// If the crash happened less than 5 seconds ago, and we haven't logged it yet
				if failoverDuration < 5 && !failoverLogged {
					fmt.Printf("[%s] FAILOVER DETECTED. Duration: %.4fs\n", myGUID[:8], failoverDuration)
					logToCSV(FailoverFile, "Failover_Time", failoverDuration)
					failoverLogged = true
				}
			}
		}

		// D. Election Logic
		if myIndex == 0 {
			// I AM THE LEADER
			fmt.Printf("\n[%s] === I AM THE LEADER ===\n", myGUID[:8])
			
			// Stay leader for 15 seconds
			time.Sleep(LeaderTenure)
			
			// Simulate Crash: Write time of death and exit
			writeDeathTimestamp()
			fmt.Println("Simulating CRASH...")
			os.Exit(0) 
		} else {
			// I AM A FOLLOWER
			// Just wait a tiny bit and check again
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// --- HELPER FUNCTIONS ---

func logToCSV(filename, metricName string, value float64) {
	// Simple append to file
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil { return }
	defer f.Close()

	// Write header only if file is empty
	stat, _ := f.Stat()
	if stat.Size() == 0 {
		f.WriteString("Metric,Seconds\n")
	}

	f.WriteString(fmt.Sprintf("%s,%.4f\n", metricName, value))
}

func writeDeathTimestamp() {
	now := float64(time.Now().UnixNano()) / 1e9
	ioutil.WriteFile(DeathTimestampFile, []byte(fmt.Sprintf("%f", now)), 0644)
}

func unixFloatToTime(val float64) time.Time {
	sec := int64(val)
	nsec := int64((val - float64(sec)) * 1e9)
	return time.Unix(sec, nsec)
}

func extractSeq(nodeName string) int {
	parts := strings.Split(nodeName, "_")
	if len(parts) < 2 { return 0 }
	seq, _ := strconv.Atoi(parts[len(parts)-1])
	return seq
}