package main

import (
	"context"
	server "juong/http/internal"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	args := os.Args[1:]
	ctx, cancelCtx := context.WithCancel(context.Background())

	errChan := make(chan error)
	port, _ := strconv.Atoi(args[0])
	srv := server.NewServerHandler(port)

	httpServer := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: srv,
	}

	go func() {
		log.Println("Starting server")
		errChan <- httpServer.ListenAndServe()
		log.Println("Received", "err", errChan)
		cancelCtx()
	}()

	<-ctx.Done()
}
