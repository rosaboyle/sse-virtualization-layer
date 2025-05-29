package manager

import (
	"fmt"
	"log"
	"sync"
	"time"

	"virtualization-manager/pkg/redis"
	"virtualization-manager/pkg/types"

	"github.com/google/uuid"
)

type ConnectionManager struct {
	redisClient *redis.Client
	connections map[string]*types.Connection
	mutex       sync.RWMutex
	startTime   time.Time
}

func NewConnectionManager(redisClient *redis.Client) *ConnectionManager {
	cm := &ConnectionManager{
		redisClient: redisClient,
		connections: make(map[string]*types.Connection),
		startTime:   time.Now(),
	}

	// Start background processes
	go cm.startHeartbeat()
	go cm.startCleanup()

	return cm
}

// AddConnection adds a new SSE connection
func (cm *ConnectionManager) AddConnection(clientID, userID string, metadata map[string]string) *types.Connection {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	connectionID := uuid.New().String()
	connection := &types.Connection{
		ID:        connectionID,
		ClientID:  clientID,
		UserID:    userID,
		Channel:   make(chan types.SSEMessage, 100), // Buffer for messages
		Metadata:  metadata,
		CreatedAt: time.Now(),
		LastPing:  time.Now(),
		Active:    true,
	}

	cm.connections[connectionID] = connection

	// Store in Redis
	if err := cm.redisClient.StoreConnection(connection); err != nil {
		log.Printf("Failed to store connection in Redis: %v", err)
	}

	// Increment connection counter
	cm.redisClient.IncrementCounter("total_connections")

	log.Printf("Added connection: %s for client: %s", connectionID, clientID)
	return connection
}

// RemoveConnection removes an SSE connection
func (cm *ConnectionManager) RemoveConnection(connectionID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if connection, exists := cm.connections[connectionID]; exists {
		connection.Active = false
		close(connection.Channel)
		delete(cm.connections, connectionID)

		// Remove from Redis
		if err := cm.redisClient.DeleteConnection(connectionID); err != nil {
			log.Printf("Failed to delete connection from Redis: %v", err)
		}

		log.Printf("Removed connection: %s", connectionID)
	}
}

// GetConnection retrieves a connection by ID
func (cm *ConnectionManager) GetConnection(connectionID string) *types.Connection {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.connections[connectionID]
}

// GetConnectionsByClientID retrieves all connections for a client
func (cm *ConnectionManager) GetConnectionsByClientID(clientID string) []*types.Connection {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var clientConnections []*types.Connection
	for _, conn := range cm.connections {
		if conn.ClientID == clientID {
			clientConnections = append(clientConnections, conn)
		}
	}

	return clientConnections
}

// GetAllConnections returns all active connections
func (cm *ConnectionManager) GetAllConnections() []*types.Connection {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	connections := make([]*types.Connection, 0, len(cm.connections))
	for _, conn := range cm.connections {
		connections = append(connections, conn)
	}

	return connections
}

// SendToConnection sends a message to a specific connection
func (cm *ConnectionManager) SendToConnection(connectionID string, message types.SSEMessage) error {
	cm.mutex.RLock()
	connection := cm.connections[connectionID]
	cm.mutex.RUnlock()

	if connection == nil || !connection.Active {
		return ErrConnectionNotFound
	}

	select {
	case connection.Channel <- message:
		return nil
	default:
		log.Printf("Connection %s channel is full, dropping message", connectionID)
		return ErrChannelFull
	}
}

// BroadcastToClient sends a message to all connections of a client
func (cm *ConnectionManager) BroadcastToClient(clientID string, message types.SSEMessage) {
	connections := cm.GetConnectionsByClientID(clientID)
	
	for _, conn := range connections {
		if err := cm.SendToConnection(conn.ID, message); err != nil {
			log.Printf("Failed to send message to connection %s: %v", conn.ID, err)
		}
	}
}

// BroadcastToAll sends a message to all active connections
func (cm *ConnectionManager) BroadcastToAll(message types.SSEMessage) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	for connectionID := range cm.connections {
		if err := cm.SendToConnection(connectionID, message); err != nil {
			log.Printf("Failed to broadcast to connection %s: %v", connectionID, err)
		}
	}
}

// UpdateLastPing updates the last ping time for a connection
func (cm *ConnectionManager) UpdateLastPing(connectionID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if connection, exists := cm.connections[connectionID]; exists {
		connection.LastPing = time.Now()

		// Update in Redis
		if err := cm.redisClient.StoreConnection(connection); err != nil {
			log.Printf("Failed to update connection ping in Redis: %v", err)
		}
	}
}

// GetStats returns connection statistics
func (cm *ConnectionManager) GetStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	clientCount := make(map[string]int)
	for _, conn := range cm.connections {
		clientCount[conn.ClientID]++
	}

	return map[string]interface{}{
		"total_connections":  len(cm.connections),
		"unique_clients":     len(clientCount),
		"uptime_seconds":     time.Since(cm.startTime).Seconds(),
		"clients_breakdown":  clientCount,
	}
}

// startHeartbeat sends periodic heartbeat messages
func (cm *ConnectionManager) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		heartbeat := types.SSEMessage{
			Event: "heartbeat",
			Data:  map[string]interface{}{"timestamp": time.Now().Unix()},
		}

		cm.BroadcastToAll(heartbeat)

		// Update metrics
		stats := cm.GetStats()
		cm.redisClient.SetMetric("connection_stats", stats)
	}
}

// startCleanup removes stale connections
func (cm *ConnectionManager) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cm.cleanup()
	}
}

func (cm *ConnectionManager) cleanup() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	staleThreshold := 2 * time.Minute
	now := time.Now()

	for connectionID, connection := range cm.connections {
		if now.Sub(connection.LastPing) > staleThreshold {
			log.Printf("Cleaning up stale connection: %s", connectionID)
			connection.Active = false
			close(connection.Channel)
			delete(cm.connections, connectionID)

			// Remove from Redis
			cm.redisClient.DeleteConnection(connectionID)
		}
	}
}

// Shutdown gracefully shuts down the connection manager
func (cm *ConnectionManager) Shutdown() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	log.Println("Shutting down connection manager...")

	for connectionID, connection := range cm.connections {
		connection.Active = false
		close(connection.Channel)
		cm.redisClient.DeleteConnection(connectionID)
	}

	cm.connections = make(map[string]*types.Connection)
	log.Println("Connection manager shutdown complete")
}

// Custom errors
var (
	ErrConnectionNotFound = fmt.Errorf("connection not found")
	ErrChannelFull       = fmt.Errorf("connection channel is full")
)
