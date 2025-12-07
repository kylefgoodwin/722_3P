package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/go-zookeeper/zk"
)

const (
	ZKHosts      = "localhost:2181"
	ElectionPath = "/election"
)

func main() {
	skipCleanup := flag.Bool("skipCleanup", false, "Skip cleanup of existing election nodes")
	flag.Parse()

	// Connect to Zookeeper
	conn, _, err := zk.Connect([]string{ZKHosts}, time.Second*5)
	if err != nil {
		fmt.Printf("failed to connect to ZooKeeper: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to ZooKeeper. Setting up election node...")

	exists, _, _ := conn.Exists(ElectionPath)
	if exists && !*skipCleanup {
		// Clean up existing election for fresh start
		children, _, err := conn.Children(ElectionPath)
		if err == nil {
			for _, child := range children {
				conn.Delete(ElectionPath+"/"+child, -1)
			}
		}
		conn.Delete(ElectionPath, -1)
		time.Sleep(100 * time.Millisecond)
	}

	// Election node creation on first run + after clean up
	if exists, _, _ := conn.Exists(ElectionPath); !exists {
		_, err := conn.Create(ElectionPath, []byte{}, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			fmt.Printf("failed to create election node: %v\n", err)
			return
		}
		fmt.Println("Election node created successfully.")
	} else {
		fmt.Println("Election node already exists (skip cleanup mode).")
	}

	fmt.Println("Setup complete.")
}
