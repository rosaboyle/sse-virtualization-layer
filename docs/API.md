# API Documentation

## Overview

The SSE Virtualization Manager provides REST API endpoints for managing Server-Sent Events connections and serverless function invocations.

**Base URL**: `http://localhost:8080`

## Authentication

Currently, no authentication is required. This is suitable for development and trusted environments.

> **Note**: For production deployments, implement authentication middleware.

## Content Types

- **Request**: `application/json`
- **Response**: `application/json`
- **SSE Stream**: `text/event-stream`

## Rate Limiting

No rate limiting is currently implemented. Consider adding rate limiting for production use.

---

## SSE Endpoints

### Establish SSE Connection

Establishes a persistent Server-Sent Events connection for real-time communication.

**Endpoint**: `GET /sse/{clientId}`

**Parameters**:
- `clientId` (path, required): Unique identifier for the client
- `app` (query, optional): Application name
- `version` (query, optional): Application version
- Additional query parameters are stored as connection metadata

**Headers**:
```http
Accept: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**Response Headers**:
```http
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
Access-Control-Allow-Origin: *
Access-Control-Allow-Headers: Cache-Control
```

**Example Request**:
```bash
curl -N -H "Accept: text/event-stream" \
  "http://localhost:8080/sse/client-123?app=myapp&version=1.0"
```

**SSE Events**:

#### `connected` Event
Sent immediately after connection establishment.

```
event: connected
data: {
  "connection_id": "conn-uuid-123",
  "client_id": "client-123",
  "timestamp": 1640995200
}
```

#### `heartbeat` Event
Sent periodically to keep the connection alive.

```
event: heartbeat
data: {
  "timestamp": 1640995200,
  "connection_id": "conn-uuid-123"
}
```

#### `function_response` Event
Sent when a serverless function execution completes.

```
event: function_response
data: {
  "request_id": "req-uuid-456",
  "function_name": "echo",
  "success": true,
  "duration": 150,
  "data": {
    "result": "function output"
  },
  "timestamp": 1640995200
}
```

**Error Response**:
```
event: error
data: {
  "error": "Connection failed",
  "code": "CONNECTION_ERROR",
  "timestamp": 1640995200
}
```

---

## Function Management

### Register Function

Registers a new serverless function in the system.

**Endpoint**: `POST /admin/functions`

**Request Body**:
```json
{
  "name": "function-name",
  "endpoint": "https://api.example.com/webhook",
  "method": "POST",
  "timeout": "30s",
  "description": "Function description",
  "headers": {
    "Authorization": "Bearer token",
    "Content-Type": "application/json"
  }
}
```

**Field Descriptions**:
- `name` (string, required): Unique function identifier
- `endpoint` (string, required): HTTP endpoint URL
- `method` (string, required): HTTP method (GET, POST, PUT, DELETE)
- `timeout` (string, optional): Timeout duration (default: "30s")
- `description` (string, optional): Function description
- `headers` (object, optional): Custom HTTP headers

**Success Response** (201):
```json
{
  "success": true,
  "function": {
    "name": "function-name",
    "endpoint": "https://api.example.com/webhook",
    "method": "POST",
    "timeout": "30s",
    "description": "Function description",
    "is_active": true,
    "registered_at": "2024-01-01T00:00:00Z",
    "last_health_check": "2024-01-01T00:00:00Z"
  }
}
```

**Error Response** (400):
```json
{
  "success": false,
  "error": "Function with name 'function-name' already exists",
  "code": "FUNCTION_EXISTS"
}
```

**Example**:
```bash
curl -X POST http://localhost:8080/admin/functions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "echo",
    "endpoint": "https://httpbin.org/post",
    "method": "POST",
    "timeout": "10s",
    "description": "Echo service for testing"
  }'
```

### Get Functions

Retrieves all registered functions and their status.

**Endpoint**: `GET /admin/functions`

**Success Response** (200):
```json
{
  "success": true,
  "count": 2,
  "functions": [
    {
      "name": "echo",
      "endpoint": "https://httpbin.org/post",
      "method": "POST",
      "timeout": "30s",
      "description": "Echo service",
      "is_active": true,
      "registered_at": "2024-01-01T00:00:00Z",
      "last_health_check": "2024-01-01T00:05:00Z",
      "health_status": "healthy"
    },
    {
      "name": "webhook",
      "endpoint": "https://api.example.com/webhook",
      "method": "POST",
      "timeout": "30s",
      "description": "Webhook handler",
      "is_active": false,
      "registered_at": "2024-01-01T00:00:00Z",
      "last_health_check": "2024-01-01T00:04:00Z",
      "health_status": "unhealthy"
    }
  ]
}
```

**Example**:
```bash
curl http://localhost:8080/admin/functions
```

---

## Function Invocation

### Invoke Function

Invokes a registered serverless function with optional async response via SSE.

**Endpoint**: `POST /invoke/{functionName}`

**Parameters**:
- `functionName` (path, required): Name of the registered function

**Request Body**:
```json
{
  "payload": {
    "key": "value",
    "data": "any json data"
  },
  "client_id": "client-123",
  "async": true,
  "timeout": "30s"
}
```

**Field Descriptions**:
- `payload` (object, required): Data to send to the function
- `client_id` (string, optional): Client ID for async response via SSE
- `async` (boolean, optional): If true, response sent via SSE (default: false)
- `timeout` (string, optional): Override function timeout

**Synchronous Response** (200):
```json
{
  "success": true,
  "request_id": "req-uuid-123",
  "function_name": "echo",
  "duration": 150,
  "data": {
    "result": "function response data"
  }
}
```

**Asynchronous Response** (202):
```json
{
  "success": true,
  "request_id": "req-uuid-123",
  "message": "Function invoked, response will be sent via SSE",
  "client_id": "client-123"
}
```

**Error Response** (404):
```json
{
  "success": false,
  "error": "Function 'unknown-function' not found",
  "code": "FUNCTION_NOT_FOUND"
}
```

**Error Response** (500):
```json
{
  "success": false,
  "error": "Function execution failed: timeout",
  "code": "EXECUTION_ERROR",
  "request_id": "req-uuid-123"
}
```

**Examples**:

**Synchronous Invocation**:
```bash
curl -X POST http://localhost:8080/invoke/echo \
  -H "Content-Type: application/json" \
  -d '{
    "payload": {"message": "Hello World"},
    "async": false
  }'
```

**Asynchronous Invocation**:
```bash
curl -X POST http://localhost:8080/invoke/echo \
  -H "Content-Type: application/json" \
  -d '{
    "payload": {"message": "Hello World"},
    "client_id": "client-123",
    "async": true
  }'
```

---

## Administrative Endpoints

### Health Check

Returns the health status of the service and its dependencies.

**Endpoint**: `GET /admin/health`

**Success Response** (200):
```json
{
  "status": "healthy",
  "timestamp": 1640995200,
  "uptime": 3600000000000,
  "version": "1.0.0",
  "active_connections": 5,
  "registered_functions": 3,
  "redis_status": "connected",
  "memory_usage": {
    "alloc": 1048576,
    "total_alloc": 5242880,
    "sys": 8388608
  }
}
```

**Unhealthy Response** (503):
```json
{
  "status": "unhealthy",
  "timestamp": 1640995200,
  "errors": [
    "Redis connection failed",
    "High memory usage"
  ]
}
```

**Example**:
```bash
curl http://localhost:8080/admin/health
```

### Get Connections

Returns information about active SSE connections.

**Endpoint**: `GET /admin/connections`

**Success Response** (200):
```json
{
  "success": true,
  "connections": [
    {
      "id": "conn-uuid-123",
      "client_id": "client-123",
      "user_id": "user-456",
      "metadata": {
        "app": "myapp",
        "version": "1.0"
      },
      "created_at": "2024-01-01T00:00:00Z",
      "last_ping": "2024-01-01T00:05:00Z",
      "active": true
    }
  ],
  "stats": {
    "total": 10,
    "active": 8,
    "inactive": 2,
    "average_duration": 1800
  }
}
```

**Example**:
```bash
curl http://localhost:8080/admin/connections
```

---

## JavaScript/Browser SDK

```javascript
// Establish SSE connection
const eventSource = new EventSource('/sse/my-client?app=webapp');

eventSource.addEventListener('function_response', (event) => {
  const response = JSON.parse(event.data);
  console.log('Function result:', response.data);
});

// Invoke function
fetch('/invoke/echo', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    payload: { message: 'Hello!' },
    client_id: 'my-client',
    async: true
  })
});
```
