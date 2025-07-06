# Go Leader Election with Etcd v2

This repository contains a simple Go application that demonstrates a distributed leader election pattern using etcd's v2 REST API. Each instance of the application runs an HTTP server that will either be a `Leader` or a `Follower`.

The primary goal of this project is to provide a clear, educational example of how leader election can be implemented from first principles using HTTP calls, without relying on a specific etcd client library.


## Features

- **Distributed Leader Election**: Multiple nodes can be run simultaneously, and they will coordinate to elect a single leader.
- **Automatic Failover**: If the current leader process is terminated or fails to communicate with etcd, its lease will expire, and the remaining follower nodes will automatically elect a new leader.
- **Simple HTTP Interface**: Each node exposes an HTTP endpoint to check its current role (`Leader` or `Follower`) and discover the address of the current leader.
- **Pure Go Standard Library**: The implementation relies only on Go's built-in `net/http` package for all communication with the etcd cluster.

## How It Works

The leader election logic is based on etcd's key-value store features, specifically atomic compare-and-swap operations and keys with a Time-To-Live (TTL).

1.  **Startup**: Every node starts in the `Follower` state.
2.  **Acquisition Attempt**: A follower attempts to become the leader by trying to create a specific key (`/v2/keys/leader`) in etcd. It uses the `prevExist=false` query parameter, which makes the operation atomic. This guarantees that only the first node to send the request will succeed in creating the key.
3.  **Becoming Leader**: If the key creation is successful (HTTP `201 Created`), the node transitions to the `Leader` role.
4.  **Maintaining Leadership**: The leader is responsible for periodically sending a request to etcd to refresh the TTL on the leader key. This acts as a "heartbeat," signaling that it is still alive.
5.  **Follower State**: If a follower's attempt to acquire the lock fails (because the key already exists), it fetches the leader's information from the key's value and continues to monitor it.
6.  **Failover**: If the leader process crashes or is partitioned from the network, it will fail to refresh the TTL. Once the TTL expires, etcd automatically deletes the key. The remaining followers will detect that the key is gone on their next check and will begin a new election by attempting to acquire the lock again.


## Prerequisites

- **Go**: Version 1.18 or later.
- **Etcd**: A running etcd instance. You can easily start one using Docker:
  ```sh
  docker run -d -p 2379:2379 --name etcd-gcr-v3.5.0 gcr.io/etcd-development/etcd:v3.5.0 /usr/local/bin/etcd --advertise-client-urls [http://0.0.0.0:2379](http://0.0.0.0:2379) --listen-client-urls [http://0.0.0.0:2379](http://0.0.0.0:2379)
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
        go run ./cmd/server 8080
        ```
    * **Terminal 2:**
        ```sh
        go run ./cmd/server 8081
        ```
    * **Terminal 3:**
        ```sh
        go run ./cmd/server 8082
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
    - Watch the logs in the other terminals. Within about 10-15 seconds, you will see one of the followers detect that the leader key has expired and transition to become the new leader.
    - Verify the new leader by curling the endpoints again.

## Code Structure

-   `main.go`: The entry point of the application. It is responsible for parsing command-line arguments (the port), setting up the HTTP server, and starting the leader election goroutine.
-   `server/server.go`: Contains the core logic for the leader election process.
    -   `ServerHandler`: The main struct that holds the state of a node (its role, port, etc.) and implements the `http.Handler` interface.
    -   `Run()`: The main loop for the state machine. It continuously tries to acquire or maintain leadership based on the node's current role.
    -   `SendLeaderRequest()`: A helper function to communicate with the etcd API to create or update the leader key.
    -   `ServeHTTP()`: The HTTP handler that responds to status requests.
