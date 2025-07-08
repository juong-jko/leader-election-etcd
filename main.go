package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	server "juong/http/internal"
)

func main() {
	// Define command-line flags
	port := flag.Int("port", 8080, "Port to listen on")
	etcdEndpoints := flag.String("etcd-endpoints", "127.0.0.1:2379", "Comma-separated list of etcd endpoints")
	flag.Parse()

	// Create a context that can be cancelled
	ctx, cancelCtx := context.WithCancel(context.Background())

	// Create a new server handler
	srv, err := server.NewServerHandler(*port, strings.Split(*etcdEndpoints, ","))
	if err != nil {
		log.Fatalf("Failed to create server handler: %v", err)
	}

	// Create a new HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: srv,
	}

	// Start the server in a new goroutine
	go func() {
		log.Println("Starting server on port", *port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe failed: %v", err)
		}
	}()

	// Wait for a termination signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutdown signal received, shutting down...")

	// Shutdown the server gracefully
	srv.Shutdown()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	cancelCtx()
}
