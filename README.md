### Assignment 3: Configuration Box
## Kyle Goodwin and Sam Harris

## To run:

    Download Apache Zookeeper
    Install kazoo to be able to talk to Zookeeper: $ pip3 install kazoo (or just pip if pip3 fails)
    Navigate to the /bin folder and run $ $/zkServer.sh start
    Once zookeeper is started (Should say "Starting zookeeper ... STARTED"), in a new terminal, run ./run_tests.sh

    It will take a little bit of time to run. The output files are cold_start_data.csv and failover_data.csv, which is what
    we used to create out report.

## AI Usage

    We leveraged Claude Sonnet 4.5 and Gemini 3 to get an idea of what libraries we needed to use to complete this project.
    Using prompts like "How can I get my local python environment to talk to Apache Zookeeper?" we decided to use Kazoo to 
    do this, and used built in libraries like "csv" and "time" to collect and store the report data. 

    The project explecitally mentioned following the "Leader Election Recipe" given by the Zookeeper documentation. We worked with AI to generate the specific logic where each node sorts the current children and sets a "DataWatch" only on the node immediately preceding it. This ensures that when a leader creashes, only one node (the one following it) wakes up to take over, rather than spiking the network with requests from all nodes. Using more specific prompts like "How do I implement the ZooKeeper Leader Election recipe in Python using the kazoo library?" was extremely helpful and gave tons of suggestions and places to start.