### Assignment 3: Configuration Box
## Kyle Goodwin and Sam Harris

## To run:

    Download Apache Zookeeper
    Navigate to the /bin folder and run $ $/zkServer.sh start
    Once zookeeper is started (Should say "Starting zookeeper ... STARTED"), in a new terminal, run ./run_tests.sh

    It will take a little bit of time to run. The output files are cold_start_data.csv and failover_data.csv, which is what
    we used to create out report.

## Test.go run

Added connmngr, main, and test packages. This splits up everything that needs to be run in the test. Now you can just start zookeeper in a terminal and in a seperate on you can run test/test.go and it will do everything. Very similar to run_tests.sh but I was struggling with the bash stuff. 

connmngr is just its own process to set up the elections node in between every test to avoid race conditions when each node was doing it seperatly. 

## AI Usage

    We leveraged Claude Sonnet 4.5 and Gemini 3 to get an idea of what libraries we needed to use to complete this project.
    Using prompts like "How can I get my local go environment to talk to Apache Zookeeper?" we found libraries like "encoding/csv" and "time" that were crucial to our success.