package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type Role int

const (
	Leader = iota
	Follower
)

type ServerHandler struct {
	client *clientv3.Client
	role   Role

	port       int
	leaderPort int

	session *concurrency.Session
}

func (s *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.role == Leader {
		fmt.Fprintf(w, "I am the leader")
	} else {
		fmt.Fprintf(w, "I am a follower and the leader is on port %d", s.leaderPort)
	}
}

func (s *ServerHandler) Run() {
	for {
		sess, err := concurrency.NewSession(s.client, concurrency.WithTTL(5)) // Set a shorter TTL for faster leader re-election
		if err != nil {
			log.Printf("Error creating session: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		s.session = sess
		e := concurrency.NewElection(sess, "/leader-election/")
		ctx := context.Background()

		// Attempt to campaign for leadership with a timeout.
		// This prevents followers from getting stuck indefinitely if a leader already exists.
		campaignCtx, cancelCampaign := context.WithTimeout(ctx, 2*time.Second) // Try to campaign for 2 seconds
		campaignErr := e.Campaign(campaignCtx, strconv.Itoa(s.port))
		cancelCampaign() // Release resources associated with this context

		if campaignErr == nil {
			// Successfully became the leader
			log.Println("Successfully became the leader")
			s.role = Leader
			log.Printf("Server role changed to: %v", s.role)
			s.observeLoop(ctx, e) // Leader also observes its own status
		} else {
			// Campaign failed (either timeout, or another error).
			// Assume there's an existing leader or a temporary issue.
			s.role = Follower
			log.Printf("Campaign failed or timed out (%v). Becoming follower on port %d.", campaignErr, s.port)

			// Try to get the current leader's info
			resp, leaderErr := e.Leader(ctx)
			if leaderErr == nil && len(resp.Kvs) > 0 {
				leaderPort, parseErr := strconv.Atoi(string(resp.Kvs[0].Value))
				if parseErr != nil {
					log.Printf("Error parsing leader port from existing leader: %v", parseErr)
				} else {
					s.leaderPort = leaderPort
					log.Printf("Follower found existing leader on port: %d", s.leaderPort)
				}
			} else {
				log.Printf("Could not find existing leader after failed campaign: %v", leaderErr)
			}

			// Now, continuously observe for changes in leadership
			timeoutCtx, _ := context.WithTimeout(ctx, 5*time.Second)
			s.observeLoop(timeoutCtx, e)
		}
		time.Sleep(1 * time.Second) // Small delay before retrying the whole loop
	}
}

// observeLoop handles continuous observation of the leader status
func (s *ServerHandler) observeLoop(ctx context.Context, e *concurrency.Election) {
	fmt.Println("Observing")
	observeChan := e.Observe(ctx)
	for resp := range observeChan {
		if len(resp.Kvs) == 0 {
			log.Println("Leadership lost or session expired. Exiting observation loop.")
			s.role = Follower // Ensure role is follower if leadership is lost
			log.Printf("Server role changed to: %v", s.role)
			return // Exit this function, allowing the main Run loop to re-evaluate
		}
		leaderPort, err := strconv.Atoi(string(resp.Kvs[0].Value))
		if err != nil {
			log.Printf("Error parsing leader port during observation: %v", err)
		} else {
			s.leaderPort = leaderPort
			log.Printf("Follower updated leader port to: %d", s.leaderPort)
		}
	}
}

func NewServerHandler(port int, etcdEndpoints []string) (*ServerHandler, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating etcd client: %w", err)
	}

	handler := &ServerHandler{
		client: cli,
		port:   port,
		role:   Follower,
	}

	go handler.Run()

	return handler, nil
}

func (s *ServerHandler) Shutdown() {
	if s.client != nil {
		s.client.Close()
	}

	if s.session != nil {
		s.session.Close()
	}
}
