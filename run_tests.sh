#!/bin/bash
# run_tests.sh

echo "Compiling Go binary..."
go build -o election_node election.go

if [ $? -ne 0 ]; then
    echo "Compilation failed. Exiting."
    exit 1
fi

for i in {1..20}; do
    echo "=== Run $i ==="
    
    ./election_node &
    PID1=$!
    ./election_node &
    PID2=$!
    ./election_node &
    PID3=$!

    sleep 20

    pkill -x election_node
    
    sleep 2
done

echo "Test suite completed."