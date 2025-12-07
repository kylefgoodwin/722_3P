#!/bin/bash
# run_tests.sh

echo "Compiling connection manager..."
go build -o connection_mngr ./connmngr/setup.go

if [ $? -ne 0 ]; then
    echo "Connection manager compilation failed. Exiting."
    exit 1
fi

echo "Compiling Go binary..."
go build -o election_node ./main/election.go

if [ $? -ne 0 ]; then
    echo "Compilation failed. Exiting."
    exit 1
fi

for i in {1..20}; do
    echo "=== Run $i ==="

    # Step 1: Setup fresh election node (clean up from previous run)
    ./connection_mngr
    sleep 1

    # Delete death timestamp from previous run
    rm -f last_leader_death.txt

    # Step 2: Start 3 nodes
    ./election_node --runNo $i --leadCrashTest=true &
    PID1=$!
    ./election_node --runNo $i --leadCrashTest=true &
    PID2=$!
    ./election_node --runNo $i --leadCrashTest=true &
    PID3=$!

    # Step 3 & 4: Wait for leader to crash and failover to complete
    # (5 seconds for leader tenure + 2 seconds buffer for failover detection)
    sleep 8

    # Step 5: Kill all remaining processes
    taskkill //F //IM election_node.exe //T 2>/dev/null
    
    # Wait for processes to fully terminate
    sleep 2
    
    # Verify all are dead
    tasklist | grep election_node.exe > /dev/null 2>&1
    while [ $? -eq 0 ]; do
        echo "Waiting for processes to terminate..."
        sleep 1
        tasklist | grep election_node.exe > /dev/null 2>&1
    done
    
    # Brief pause before next run
    sleep 1
done

echo "Test suite completed."
echo "Results saved to failover_data.csv (should have 40 data points from 20 runs)"