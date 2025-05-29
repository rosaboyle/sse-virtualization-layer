package registry

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"virtualization-manager/pkg/redis"
	"virtualization-manager/pkg/types"
)

type FunctionRegistry struct {
	redisClient *redis.Client
	functions   map[string]*types.Function
	mutex       sync.RWMutex
}

func NewFunctionRegistry(redisClient *redis.Client) *FunctionRegistry {
	fr := &FunctionRegistry{
		redisClient: redisClient,
		functions:   make(map[string]*types.Function),
	}

	// Load existing functions from Redis
	fr.loadFunctionsFromRedis()

	// Start health checking
	go fr.startHealthCheck()

	return fr
}

// RegisterFunction registers a new serverless function
func (fr *FunctionRegistry) RegisterFunction(w http.ResponseWriter, r *http.Request) {
	var function types.Function
	if err := json.NewDecoder(r.Body).Decode(&function); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set default values
	if function.Method == "" {
		function.Method = "POST"
	}
	if function.Timeout == 0 {
		function.Timeout = 30 * time.Second
	}

	function.IsActive = true
	function.CreatedAt = time.Now()
	function.UpdatedAt = time.Now()

	if err := fr.AddFunction(&function); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(function)
}

// AddFunction adds a function to the registry
func (fr *FunctionRegistry) AddFunction(function *types.Function) error {
	fr.mutex.Lock()
	defer fr.mutex.Unlock()

	fr.functions[function.Name] = function

	// Store in Redis
	if err := fr.redisClient.StoreFunction(function); err != nil {
		return fmt.Errorf("failed to store function in Redis: %v", err)
	}

	log.Printf("Registered function: %s at %s", function.Name, function.Endpoint)
	return nil
}

// GetFunction retrieves a function by name
func (fr *FunctionRegistry) GetFunction(name string) (*types.Function, error) {
	fr.mutex.RLock()
	defer fr.mutex.RUnlock()

	if function, exists := fr.functions[name]; exists {
		return function, nil
	}

	return nil, fmt.Errorf("function %s not found", name)
}

// GetFunctions returns all registered functions
func (fr *FunctionRegistry) GetFunctions(w http.ResponseWriter, r *http.Request) {
	fr.mutex.RLock()
	functions := make([]*types.Function, 0, len(fr.functions))
	for _, fn := range fr.functions {
		functions = append(functions, fn)
	}
	fr.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"functions": functions,
		"count":     len(functions),
	})
}

// RemoveFunction removes a function from the registry
func (fr *FunctionRegistry) RemoveFunction(name string) error {
	fr.mutex.Lock()
	defer fr.mutex.Unlock()

	if _, exists := fr.functions[name]; !exists {
		return fmt.Errorf("function %s not found", name)
	}

	delete(fr.functions, name)

	// Remove from Redis
	if err := fr.redisClient.DeleteFunction(name); err != nil {
		return fmt.Errorf("failed to delete function from Redis: %v", err)
	}

	log.Printf("Removed function: %s", name)
	return nil
}

// UpdateFunctionStatus updates the active status of a function
func (fr *FunctionRegistry) UpdateFunctionStatus(name string, isActive bool) error {
	fr.mutex.Lock()
	defer fr.mutex.Unlock()

	function, exists := fr.functions[name]
	if !exists {
		return fmt.Errorf("function %s not found", name)
	}

	function.IsActive = isActive
	function.UpdatedAt = time.Now()

	// Update in Redis
	if err := fr.redisClient.StoreFunction(function); err != nil {
		return fmt.Errorf("failed to update function in Redis: %v", err)
	}

	log.Printf("Updated function %s status to %v", name, isActive)
	return nil
}

// GetActiveFunctions returns only active functions
func (fr *FunctionRegistry) GetActiveFunctions() map[string]*types.Function {
	fr.mutex.RLock()
	defer fr.mutex.RUnlock()

	activeFunctions := make(map[string]*types.Function)
	for name, function := range fr.functions {
		if function.IsActive {
			activeFunctions[name] = function
		}
	}

	return activeFunctions
}

// loadFunctionsFromRedis loads functions from Redis on startup
func (fr *FunctionRegistry) loadFunctionsFromRedis() {
	functions, err := fr.redisClient.GetAllFunctions()
	if err != nil {
		log.Printf("Failed to load functions from Redis: %v", err)
		return
	}

	fr.mutex.Lock()
	defer fr.mutex.Unlock()

	for _, function := range functions {
		fr.functions[function.Name] = function
	}

	log.Printf("Loaded %d functions from Redis", len(functions))
}

// startHealthCheck performs periodic health checks on registered functions
func (fr *FunctionRegistry) startHealthCheck() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		fr.performHealthCheck()
	}
}

func (fr *FunctionRegistry) performHealthCheck() {
	activeFunctions := fr.GetActiveFunctions()

	for name, function := range activeFunctions {
		go fr.checkFunctionHealth(name, function)
	}
}

func (fr *FunctionRegistry) checkFunctionHealth(name string, function *types.Function) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create a simple health check request
	req, err := http.NewRequest("GET", function.Endpoint+"/health", nil)
	if err != nil {
		log.Printf("Failed to create health check request for %s: %v", name, err)
		return
	}

	// Add custom headers if any
	for key, value := range function.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Health check failed for function %s: %v", name, err)
		fr.UpdateFunctionStatus(name, false)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Function is healthy
		if !function.IsActive {
			log.Printf("Function %s is back online", name)
			fr.UpdateFunctionStatus(name, true)
		}
	} else {
		log.Printf("Function %s returned unhealthy status: %d", name, resp.StatusCode)
		fr.UpdateFunctionStatus(name, false)
	}
}

// GetStats returns registry statistics
func (fr *FunctionRegistry) GetStats() map[string]interface{} {
	fr.mutex.RLock()
	defer fr.mutex.RUnlock()

	totalFunctions := len(fr.functions)
	activeFunctions := 0

	for _, function := range fr.functions {
		if function.IsActive {
			activeFunctions++
		}
	}

	return map[string]interface{}{
		"total_functions":  totalFunctions,
		"active_functions": activeFunctions,
		"inactive_functions": totalFunctions - activeFunctions,
	}
}
