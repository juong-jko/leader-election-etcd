package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Role int

const (
	Leader = iota
	Follower
)

type ServerHandler struct {
	httpClient *http.Client
	role       Role

	port       int
	leaderPort int
}

type NodeData struct {
	CreatedIndex  int    `json:"createdIndex"`
	Key           string `json:"key"`
	ModifiedIndex int    `json:"modifiedIndex"`
	Value         string `json:"value"`
}

type Response struct {
	Action string   `json:"action"`
	Node   NodeData `json:"node"`
}

type LeaderRequest struct {
	LeaderPort int `json:"value"`
}

const (
	baseUrl = "http://127.0.0.1:2379/v2/"
)

func (s *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.role == Leader {
		fmt.Fprintf(w, "I am the leader")
	} else {
		fmt.Fprintf(w, "I am a follower and the leader is on port %d", s.leaderPort)
	}
}

func (s *ServerHandler) SendLeaderRequest(prevExist bool) *http.Response {
	// Create a new PUT request
	leaderData := url.Values{}
	leaderData.Set("value", strconv.Itoa(s.port))
	leaderData.Set("ttl", strconv.Itoa(15))

	compareAndSwapURL := baseUrl + fmt.Sprintf("keys/leader?prevExist=%v", prevExist)
	req, err := http.NewRequest(http.MethodPut, compareAndSwapURL, strings.NewReader(leaderData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	compareAndSwapResp, err := s.httpClient.Do(req)
	if err != nil {
		log.Fatalf("Error sending PUT request: %v", err)
	}

	log.Printf("Compare and Swap Resp: %s\n", compareAndSwapResp.Status)

	return compareAndSwapResp
}

func (s *ServerHandler) Run() {
	for {
		switch s.role {
		case Follower:
			{
				compareAndSwapResp := s.SendLeaderRequest(false)
				defer compareAndSwapResp.Body.Close()
				if compareAndSwapResp.StatusCode == http.StatusCreated {
					// Promoted to leader
					s.role = Leader
					break
				}

				log.Println("Fetching Info about leader")
				resp, err := s.httpClient.Get(baseUrl + "keys/leader")
				if err != nil {
					log.Printf("Error: %v\n", err)
				}

				var data Response
				err = json.NewDecoder(resp.Body).Decode(&data)
				if err != nil {
					log.Printf("Error parsing JSON: %v\n", err)
				} else {
					log.Printf("Status: %s\n", resp.Status)
					log.Printf("Response: %v\n", data)
					s.leaderPort, _ = strconv.Atoi(data.Node.Value)
				}

				defer resp.Body.Close()
			}
		case Leader:
			{
				resp := s.SendLeaderRequest(true)
				defer resp.Body.Close()

				// We may have lost leadership
				if resp.StatusCode != http.StatusOK {
					log.Printf("Failed to refresh leader TTL (status: %s). Demoting to follower.\n", resp.Status)
					s.role = Follower
					continue
				}
				log.Printf("Leader TTL Request Success: %s", resp.Status)
			}
		}
		<-time.After(10 * time.Second)
	}
}

func NewServerHandler(port int) http.Handler {
	handler := &ServerHandler{
		httpClient: &http.Client{},
		port:       port,
		role:       Follower,
	}

	go handler.Run()

	return handler
}
