package types

import (
	"time"
)

// Connection represents an active SSE connection
type Connection struct {
	ID        string            `json:"id"`
	ClientID  string            `json:"client_id"`
	UserID    string            `json:"user_id,omitempty"`
	Channel   chan SSEMessage   `json:"-"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	LastPing  time.Time         `json:"last_ping"`
	Active    bool              `json:"active"`
}

// SSEMessage represents a message sent over SSE
type SSEMessage struct {
	ID    string      `json:"id,omitempty"`
	Event string      `json:"event,omitempty"`
	Data  interface{} `json:"data"`
	Retry int         `json:"retry,omitempty"`
}

// Function represents a registered serverless function
type Function struct {
	Name        string            `json:"name"`
	Endpoint    string            `json:"endpoint"`
	Method      string            `json:"method"`
	Timeout     time.Duration     `json:"timeout"`
	Headers     map[string]string `json:"headers"`
	Description string            `json:"description"`
	IsActive    bool              `json:"is_active"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// InvocationRequest represents a function invocation request
type InvocationRequest struct {
	FunctionName string                 `json:"function_name"`
	Payload      map[string]interface{} `json:"payload"`
	ClientID     string                 `json:"client_id,omitempty"`
	Async        bool                   `json:"async"`
	Timeout      int                    `json:"timeout,omitempty"`
}

// InvocationResponse represents a function invocation response
type InvocationResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Duration  int64       `json:"duration_ms"`
	RequestID string      `json:"request_id"`
}

// HealthStatus represents system health status
type HealthStatus struct {
	Status           string            `json:"status"`
	ActiveConnections int              `json:"active_connections"`
	RegisteredFunctions int            `json:"registered_functions"`
	RedisConnected   bool              `json:"redis_connected"`
	Uptime           time.Duration     `json:"uptime"`
	Metrics          map[string]interface{} `json:"metrics"`
}
