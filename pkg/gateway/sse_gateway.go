package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"virtualization-manager/pkg/manager"
	"virtualization-manager/pkg/registry"
	"virtualization-manager/pkg/types"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type SSEGateway struct {
	connectionManager *manager.ConnectionManager
	functionRegistry  *registry.FunctionRegistry
	startTime         time.Time
}

func NewSSEGateway(connectionManager *manager.ConnectionManager, functionRegistry *registry.FunctionRegistry) *SSEGateway {
	return &SSEGateway{
		connectionManager: connectionManager,
		functionRegistry:  functionRegistry,
		startTime:         time.Now(),
	}
}

// HandleSSEConnection handles incoming SSE connection requests
func (sg *SSEGateway) HandleSSEConnection(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Get client ID from URL params
	vars := mux.Vars(r)
	clientID := vars["clientId"]
	if clientID == "" {
		http.Error(w, "Client ID is required", http.StatusBadRequest)
		return
	}

	// Extract metadata from query parameters
	metadata := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			metadata[key] = values[0]
		}
	}

	userID := r.Header.Get("X-User-ID")

	// Create new connection
	connection := sg.connectionManager.AddConnection(clientID, userID, metadata)
	defer sg.connectionManager.RemoveConnection(connection.ID)

	// Send welcome message
	welcomeMsg := types.SSEMessage{
		ID:    uuid.New().String(),
		Event: "connected",
		Data: map[string]interface{}{
			"connection_id": connection.ID,
			"client_id":     clientID,
			"timestamp":     time.Now().Unix(),
			"message":       "Connected to SSE Virtualization Manager",
		},
	}

	sg.writeSSEMessage(w, welcomeMsg)

	// Listen for client disconnect
	notify := w.(http.CloseNotifier).CloseNotify()

	// Message processing loop
	for {
		select {
		case <-notify:
			// Client disconnected
			log.Printf("Client %s disconnected", clientID)
			return

		case message, ok := <-connection.Channel:
			if !ok {
				// Channel closed
				return
			}

			// Send message to client
			sg.writeSSEMessage(w, message)

			// Update last ping
			sg.connectionManager.UpdateLastPing(connection.ID)

		case <-time.After(30 * time.Second):
			// Send heartbeat if no messages
			heartbeat := types.SSEMessage{
				Event: "heartbeat",
				Data:  map[string]interface{}{"timestamp": time.Now().Unix()},
			}
			sg.writeSSEMessage(w, heartbeat)
		}
	}
}

// InvokeFunction handles function invocation requests
func (sg *SSEGateway) InvokeFunction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	functionName := vars["functionName"]

	var request types.InvocationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	request.FunctionName = functionName

	// Get function details
	function, err := sg.functionRegistry.GetFunction(functionName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Function not found: %s", functionName), http.StatusNotFound)
		return
	}

	if !function.IsActive {
		http.Error(w, fmt.Sprintf("Function %s is not active", functionName), http.StatusServiceUnavailable)
		return
	}

	// Generate request ID
	requestID := uuid.New().String()
	startTime := time.Now()

	// Prepare function invocation
	response, err := sg.invokeFunctionEndpoint(function, request, requestID)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		response = &types.InvocationResponse{
			Success:   false,
			Error:     err.Error(),
			Duration:  duration,
			RequestID: requestID,
		}
	} else {
		response.Duration = duration
		response.RequestID = requestID
	}

	// If client ID is provided, send result via SSE
	if request.ClientID != "" {
		message := types.SSEMessage{
			ID:    requestID,
			Event: "function_response",
			Data:  response,
		}

		sg.connectionManager.BroadcastToClient(request.ClientID, message)
	}

	// Always return HTTP response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// invokeFunctionEndpoint invokes the actual serverless function
func (sg *SSEGateway) invokeFunctionEndpoint(function *types.Function, request types.InvocationRequest, requestID string) (*types.InvocationResponse, error) {
	// Prepare payload
	payload, err := json.Marshal(request.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(function.Method, function.Endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Request-ID", requestID)
	httpReq.Header.Set("X-Client-ID", request.ClientID)

	// Add custom function headers
	for key, value := range function.Headers {
		httpReq.Header.Set(key, value)
	}

	// Configure HTTP client with timeout
	timeout := function.Timeout
	if request.Timeout > 0 {
		timeout = time.Duration(request.Timeout) * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
	}

	// Make the request
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("function invocation failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Handle different content types
	var responseData interface{}
	contentType := resp.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		if err := json.Unmarshal(responseBody, &responseData); err != nil {
			// If JSON parsing fails, return as string
			responseData = string(responseBody)
		}
	} else {
		responseData = string(responseBody)
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &types.InvocationResponse{
		Success: success,
		Data:    responseData,
	}, nil
}

// writeSSEMessage writes an SSE message to the response writer
func (sg *SSEGateway) writeSSEMessage(w http.ResponseWriter, message types.SSEMessage) {
	if message.ID != "" {
		fmt.Fprintf(w, "id: %s\n", message.ID)
	}

	if message.Event != "" {
		fmt.Fprintf(w, "event: %s\n", message.Event)
	}

	// Convert data to JSON
	data, err := json.Marshal(message.Data)
	if err != nil {
		log.Printf("Failed to marshal SSE message data: %v", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n", string(data))

	if message.Retry > 0 {
		fmt.Fprintf(w, "retry: %d\n", message.Retry)
	}

	fmt.Fprintf(w, "\n")

	// Flush the data
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// GetConnections returns information about active connections
func (sg *SSEGateway) GetConnections(w http.ResponseWriter, r *http.Request) {
	connections := sg.connectionManager.GetAllConnections()
	stats := sg.connectionManager.GetStats()

	response := map[string]interface{}{
		"connections": connections,
		"stats":       stats,
		"timestamp":   time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HealthCheck returns the health status of the gateway
func (sg *SSEGateway) HealthCheck(w http.ResponseWriter, r *http.Request) {
	connectionStats := sg.connectionManager.GetStats()
	functionStats := sg.functionRegistry.GetStats()

	health := types.HealthStatus{
		Status:              "healthy",
		ActiveConnections:   len(sg.connectionManager.GetAllConnections()),
		RegisteredFunctions: functionStats["total_functions"].(int),
		RedisConnected:      true, // TODO: Implement actual Redis health check
		Uptime:              time.Since(sg.startTime),
		Metrics: map[string]interface{}{
			"connections": connectionStats,
			"functions":   functionStats,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}
