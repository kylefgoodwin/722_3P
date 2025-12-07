package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	SetupBinary    = "../connmngr/setup.go"
	ElectionBinary = "../main/election.go"
	NumProcesses   = 3
)

func main() {
	iterations := flag.Int("iterations", 5, "Number of test iterations to run")
	flag.Parse()

	fmt.Printf("Starting Leader Election Test - %d iterations\n", *iterations)
	fmt.Println("=========================================")

	for i := 1; i <= *iterations; i++ {
		fmt.Printf("\n--- RUN %d ---\n", i)
		runIteration(i)
		time.Sleep(1 * time.Second) // Brief pause between iterations
	}

	fmt.Println("\n=========================================")
	fmt.Println("All test iterations complete!")
}

func runIteration(runNo int) {
	// Step 1: Run setup to create fresh election node
	fmt.Printf("[Run %d] Step 1: Running setup (cleaning up old nodes)...\n", runNo)
	setupCmd := exec.Command("go", "run", SetupBinary)
	setupCmd.Stdout = os.Stdout
	setupCmd.Stderr = os.Stderr
	if err := setupCmd.Run(); err != nil {
		fmt.Printf("Setup failed: %v\n", err)
		return
	}
	time.Sleep(500 * time.Millisecond) // Give ZK time to settle

	// Step 2: Start 3 election processes in parallel
	fmt.Printf("[Run %d] Step 2: Starting %d election processes...\n", runNo, NumProcesses)

	var wg sync.WaitGroup
	processes := make([]*exec.Cmd, NumProcesses)

	for i := 0; i < NumProcesses; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			cmd := exec.Command("go", "run", ElectionBinary, "-runNo", fmt.Sprintf("%d", runNo))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			processes[idx] = cmd

			// Start the process
			if err := cmd.Start(); err != nil {
				fmt.Printf("Failed to start process %d: %v\n", idx, err)
				return
			}

			// Wait for it to finish (will exit after leader crashes or gets killed)
			cmd.Wait()
		}(i)
	}

	// Step 3: Wait for the leader tenure + some buffer for failover
	// Leader runs for 5 seconds, then we need time for failover detection
	fmt.Printf("[Run %d] Step 3: Waiting for leader tenure + failover...\n", runNo)
	time.Sleep(8 * time.Second)

	// Step 4: Kill any remaining processes
	fmt.Printf("[Run %d] Step 4: Cleaning up remaining processes...\n", runNo)
	for i, cmd := range processes {
		if cmd != nil && cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				// Process might have already exited
				fmt.Printf("Process %d already terminated\n", i)
			}
		}
	}

	// Wait for all goroutines to finish
	wg.Wait()

	fmt.Printf("[Run %d] Complete!\n", runNo)
}
