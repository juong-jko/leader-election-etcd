package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	server "juong/http/internal"
)

func main() {
	args := os.Args[1:]
	ctx, cancelCtx := context.WithCancel(context.Background())

	port, _ := strconv.Atoi(args[0])
	srv := server.NewServerHandler(port)

	httpServer := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: srv,
	}

	go func() {
		log.Println("Starting server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe failed: %v", err)
		}
	}()

	// Wait for termination signal
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
