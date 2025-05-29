package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"virtualization-manager/pkg/config"
	"virtualization-manager/pkg/gateway"
	"virtualization-manager/pkg/manager"
	"virtualization-manager/pkg/registry"
	"virtualization-manager/pkg/redis"

	"github.com/gorilla/mux"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize Redis client
	redisClient := redis.NewClient(cfg.Redis)

	// Initialize core components
	connectionManager := manager.NewConnectionManager(redisClient)
	functionRegistry := registry.NewFunctionRegistry(redisClient)
	sseGateway := gateway.NewSSEGateway(connectionManager, functionRegistry)

	// Setup HTTP router
	router := mux.NewRouter()
	
	// SSE endpoint
	router.HandleFunc("/sse/{clientId}", sseGateway.HandleSSEConnection).Methods("GET")
	
	// Admin endpoints
	router.HandleFunc("/admin/connections", sseGateway.GetConnections).Methods("GET")
	router.HandleFunc("/admin/health", sseGateway.HealthCheck).Methods("GET")
	router.HandleFunc("/admin/functions", functionRegistry.GetFunctions).Methods("GET")
	router.HandleFunc("/admin/functions", functionRegistry.RegisterFunction).Methods("POST")
	
	// Function invocation endpoint
	router.HandleFunc("/invoke/{functionName}", sseGateway.InvokeFunction).Methods("POST")

	// Enable CORS
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			if r.Method == "OPTIONS" {
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})

	// Start server
	log.Printf("Starting SSE Virtualization Manager on port %s", cfg.Server.Port)
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Handle graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down gracefully...")
	connectionManager.Shutdown()
	log.Println("Server stopped")
}
