import time
import sys
import os
import uuid
import csv
from kazoo.client import KazooClient
from kazoo.exceptions import ConnectionLoss, NoNodeError

# --- CONFIGURATION ---
ZK_HOSTS = 'localhost:2181'
ELECTION_PATH = '/election'
DEATH_TIMESTAMP_FILE = 'last_leader_death.txt'
COLD_START_FILE = 'cold_start_data.csv'
FAILOVER_FILE = 'failover_data.csv'

LEADER_TENURE = 15 

class LeaderElectionNode:
    def __init__(self):
        self.zk = KazooClient(hosts=ZK_HOSTS)
        self.zk.start()
        self.my_guid = str(uuid.uuid4())
        self.my_path = None
        self.start_time = time.time()
        self.is_leader = False
        self.running = True
        
    def log_to_csv(self, filename, metric_name, value):
        file_exists = os.path.isfile(filename)
        try:
            with open(filename, 'a', newline='') as f:
                writer = csv.writer(f)
                if not file_exists:
                    writer.writerow(['Metric', 'Seconds'])
                writer.writerow([metric_name, f"{value:.4f}"])
        except Exception as e:
            print(f"[{self.my_guid[:8]}] Error writing to CSV: {e}")

    def volunteer(self):
        try:
            self.zk.ensure_path(ELECTION_PATH)
            self.my_path = self.zk.create(
                f"{ELECTION_PATH}/guid-n_", 
                ephemeral=True, 
                sequence=True
            )
            print(f"[{self.my_guid[:8]}] My Path: {self.my_path}")
            self.check_leadership()
        except ConnectionLoss as e:
            print(f"[{self.my_guid[:8]}] Connection lost during volunteer: {e}")
            sys.exit(1)

    def check_leadership(self):
        if not self.running:
            return
            
        try:
            children = self.zk.get_children(ELECTION_PATH)
        except NoNodeError:
            print(f"[{self.my_guid[:8]}] Election path disappeared, recreating...")
            self.zk.ensure_path(ELECTION_PATH)
            self.volunteer()
            return
        except ConnectionLoss as e:
            print(f"[{self.my_guid[:8]}] Connection lost: {e}")
            return

        if not children:
            print(f"[{self.my_guid[:8]}] No children found, re-volunteering...")
            self.volunteer()
            return

        sorted_children = sorted(children, key=lambda x: int(x.split("_")[-1]))
        my_node_name = self.my_path.split("/")[-1]
        
        if my_node_name not in sorted_children:
            print(f"[{self.my_guid[:8]}] My node disappeared! Re-volunteering...")
            self.volunteer()
            return

        my_index = sorted_children.index(my_node_name)

        # DATA COLLECTION: COLD START
        if len(sorted_children) > 0 and not hasattr(self, 'cold_start_logged'):
            leader_found_time = time.time()
            duration = leader_found_time - self.start_time
            
            if duration < 30:
                print(f"[{self.my_guid[:8]}] Found leader! Cold Start Time: {duration:.4f}s")
                self.log_to_csv(COLD_START_FILE, "Cold_Start_Time", duration)
                self.cold_start_logged = True

        # ELECTION LOGIC
        if my_index == 0:
            self.become_leader()
        else:
            predecessor = sorted_children[my_index - 1]
            predecessor_path = f"{ELECTION_PATH}/{predecessor}"
            print(f"[{self.my_guid[:8]}] I am follower. Watching predecessor: {predecessor}")
            
            if self.zk.exists(predecessor_path, watch=self.predecessor_deleted):
                pass
            else:
                print(f"[{self.my_guid[:8]}] Predecessor already gone!")
                self.check_leadership()

    def predecessor_deleted(self, event):
        print(f"[{self.my_guid[:8]}] Predecessor deleted event received!")
        self.check_leadership()

    def become_leader(self):
        if self.is_leader:
            return  
            
        self.is_leader = True
        print(f"\n[{self.my_guid[:8]}] === I AM THE LEADER ===")
        
        # DATA COLLECTION: FAILOVER
        if os.path.exists(DEATH_TIMESTAMP_FILE):
            try:
                with open(DEATH_TIMESTAMP_FILE, 'r') as f:
                    death_time = float(f.read().strip())
                
                failover_duration = time.time() - death_time
                
                if failover_duration < 60:
                    print(f"[{self.my_guid[:8]}] FAILOVER COMPLETE. Duration: {failover_duration:.4f}s")
                    self.log_to_csv(FAILOVER_FILE, "Failover_Time", failover_duration)
                
                os.remove(DEATH_TIMESTAMP_FILE)
            except Exception as e:
                print(f"[{self.my_guid[:8]}] Error processing failover data: {e}")

        print(f"[{self.my_guid[:8]}] Holding leadership for {LEADER_TENURE} seconds...")
        time.sleep(LEADER_TENURE)
        
        print(f"[{self.my_guid[:8]}] Simulating CRASH now...")
        try:
            with open(DEATH_TIMESTAMP_FILE, 'w') as f:
                f.write(str(time.time()))
        except Exception as e:
            print(f"[{self.my_guid[:8]}] Error writing death timestamp: {e}")
            
        self.running = False
        self.zk.stop()
        os._exit(0)

    def run_as_follower(self):
        try:
            while self.running and not self.is_leader:
                time.sleep(1)
        except KeyboardInterrupt:
            print(f"\n[{self.my_guid[:8]}] Shutting down gracefully...")
            self.cleanup()

    def cleanup(self):
        self.running = False
        try:
            if self.zk:
                self.zk.stop()
                self.zk.close()
        except Exception as e:
            print(f"[{self.my_guid[:8]}] Error during cleanup: {e}")

if __name__ == "__main__":
    node = LeaderElectionNode()
    try:
        node.volunteer()
        
        if not node.is_leader:
            node.run_as_follower()
            
    except KeyboardInterrupt:
        print("\nShutdown requested...")
        node.cleanup()
    except Exception as e:
        print(f"Unexpected error: {e}")
        node.cleanup()
        sys.exit(1)