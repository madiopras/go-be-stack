package main

import (
	"betest/internal/database"
	"betest/internal/handlers"
	"betest/internal/middleware"
	"betest/internal/routes"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Init DB
	database.InitDB()
	// kalau nanti ada CloseDB(), taruh di shutdown

	// Set public key and Redis client for middleware
	middleware.PublicKey = handlers.PublicKey
	middleware.Rdb = handlers.Rdb

	r := routes.SetupRoutes()

	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Jalankan server di goroutine
	go func() {
		log.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	// Tangkap signal OS (Ctrl+C, docker stop, kill)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Println("Shutdown signal received...")

	// Timeout shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
