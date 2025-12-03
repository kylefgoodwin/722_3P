#!/bin/bash
# run_tests.sh


for i in {1..20}; do
    echo "=== Run $i ==="
    
    sleep 2

    python3 election.py &
    PID1=$!
    python3 election.py &
    PID2=$!
    python3 election.py &
    PID3=$!

    sleep 20
    
    pkill -f election.py
    
    sleep 2
done