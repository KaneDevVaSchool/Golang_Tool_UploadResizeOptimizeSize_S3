package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"s3-upload-tool/internal/container"
)

func main() {
	ctn, err := container.NewContainer()
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}

	server := &http.Server{
		Addr:         ":" + ctn.Config.Server.Port,
		Handler:      ctn.GetServerHandler(),
		ReadTimeout:  ctn.Config.Server.ReadTimeout,
		WriteTimeout: ctn.Config.Server.WriteTimeout,
		IdleTimeout:  ctn.Config.Server.IdleTimeout,
	}

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		log.Printf("Server starting on port %s", ctn.Config.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
		serverCancel()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctn.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), ctn.Config.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	<-serverCtx.Done()
	log.Println("Server exited")
}
