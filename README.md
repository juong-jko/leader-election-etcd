# Go Leader Election with Etcd v3

This repository contains a simple Go application that demonstrates a distributed leader election pattern using etcd's v3 API.

## Features

- **Distributed Leader Election**: Multiple nodes can be run simultaneously, and they will coordinate to elect a single leader.
- **Automatic Failover**: If the current leader process is terminated or fails to communicate with etcd, its lease will expire, and the remaining follower nodes will automatically elect a new leader.
- **Simple HTTP Interface**: Each node exposes an HTTP endpoint to check its current role (`Leader` or `Follower`) and discover the address of the current leader.
- **Etcd v3 Client**: The implementation uses the official `go.etcd.io/etcd/client/v3` library.

## How It Works

The leader election logic is based on the leader election recipe from the etcd v3 API. This recipe uses leases and a simple election API to provide a robust and efficient way to handle leader election.

1.  **Startup**: Every node starts in the `Follower` state.
2.  **Campaign**: Each node starts a campaign for leadership. The first node to successfully campaign becomes the leader.
3.  **Maintaining Leadership**: The leader continuously renews its lease with etcd. This acts as a heartbeat, signaling that it is still alive.
4.  **Follower State**: Followers observe the current leader. If the leader's lease expires, the followers will start a new campaign for leadership.
5.  **Failover**: If the leader process crashes or is partitioned from the network, it will fail to renew its lease. Once the lease expires, the other nodes will detect this and one of them will be elected as the new leader.

## Prerequisites

- **Go**: Version 1.18 or later.
- **Etcd**: A running etcd instance. You can easily start one using Docker:
  ```sh
  docker run -d -p 2379:2379 --name etcd-gcr-v3.5.0 gcr.io/etcd-development/etcd:v3.5.0 /usr/local/bin/etcd --advertise-client-urls http://0.0.0.0:2379 --listen-client-urls http://0.0.0.0:2379
  ```

## How to Run

1.  **Clone the repository:**
    ```sh
    git clone <your-repo-url>
    cd <your-repo-directory>
    ```

2.  **Run multiple instances**: Open several terminal windows. In each one, run the application on a different port.

    * **Terminal 1:**
        ```sh
        go run main.go -port 8080
        ```
    * **Terminal 2:**
        ```sh
        go run main.go -port 8081
        ```
    * **Terminal 3:**
        ```sh
        go run main.go -port 8082
        ```

    The first node to start will likely become the leader. You will see log output indicating the role of each node.

3.  **Check Node Status**: Use `curl` to query the HTTP endpoint of each node.

    * **Check the leader (e.g., port 8080):**
        ```sh
        curl http://localhost:8080
        # Expected Output: I am the leader
        ```

    * **Check a follower (e.g., port 8081):**
        ```sh
        curl http://localhost:8081
        # Expected Output: I am a follower and the leader is on port 8080
        ```

4.  **Test Failover**:
    - Stop the leader process (e.g., press `Ctrl+C` in Terminal 1).
    - Watch the logs in the other terminals. You will see one of the followers detect that the leader has gone and transition to become the new leader.
    - Verify the new leader by curling the endpoints again.
