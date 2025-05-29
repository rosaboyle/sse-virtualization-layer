# SSE Virtualization Manager

A Golang-based SSE (Server-Sent Events) virtualization layer that maintains persistent connections while only invoking serverless functions on-demand. This architecture optimizes costs and improves user experience by separating connection management from function execution.

## Features

- **Persistent SSE Connections**: Maintains long-lived connections with automatic heartbeat
- **On-Demand Function Invocation**: Functions only run when needed, optimizing costs
- **Redis State Management**: Reliable state persistence and connection tracking
- **Real-time Monitoring**: Built-in health checks and metrics
- **Function Registry**: Dynamic function registration and health monitoring
- **High Performance**: Built with Go for optimal performance and concurrency

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Start the entire stack
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the stack
docker-compose down
```

### Manual Setup

1. **Start Redis:**
```bash
docker run -d --name redis -p 6379:6379 redis:7-alpine
```

2. **Install dependencies:**
```bash
go mod tidy
```

3. **Run the application:**
```bash
go run main.go
```

## API Endpoints

### SSE Connection
```
GET /sse/{clientId}
```
Establishes persistent SSE connection for a client.

**Parameters:**
- `clientId` (path): Unique identifier for the client
- `X-User-ID` (header, optional): User identifier
- Query parameters become connection metadata

**Example:**
```bash
curl -N -H "Accept: text/event-stream" \
  "http://localhost:8080/sse/client123?app=myapp&version=1.0"
```

### Function Registration
```
POST /admin/functions
```
Register a new serverless function.

**Request Body:**
```json
{
  "name": "my-function",
  "endpoint": "https://my-serverless-function.vercel.app/api/process",
  "method": "POST",
  "timeout": "30s",
  "description": "My awesome serverless function",
  "headers": {
    "Authorization": "Bearer token"
  }
}
```

### Function Invocation
```
POST /invoke/{functionName}
```
Invoke a registered function. Results can be sent via SSE or HTTP response.

**Request Body:**
```json
{
  "payload": {
    "message": "Hello, World!",
    "data": [1, 2, 3]
  },
  "client_id": "client123",
  "async": true,
  "timeout": 60
}
```

### Admin Endpoints

**Get All Connections:**
```
GET /admin/connections
```

**Health Check:**
```
GET /admin/health
```

**Get All Functions:**
```
GET /admin/functions
```

## Configuration

Set environment variables:

```bash
export PORT=8080
export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=""
```

## Architecture

```
┌─────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Clients   │◄──►│  SSE Gateway     │◄──►│ Connection Mgr  │
│             │    │                  │    │                 │
│ - Browser   │    │ - HTTP Handlers  │    │ - State Mgmt    │
│ - Mobile    │    │ - SSE Streaming  │    │ - Heartbeat     │
│ - IoT       │    │ - CORS Support   │    │ - Cleanup       │
└─────────────┘    └──────────────────┘    └─────────────────┘
                             │                        │
                             ▼                        ▼
                   ┌──────────────────┐    ┌─────────────────┐
                   │ Function Router  │◄──►│     Redis       │
                   │                  │    │                 │
                   │ - Load Balancing │    │ - Connections   │
                   │ - Health Checks  │    │ - Functions     │
                   │ - Retry Logic    │    │ - Metrics       │
                   └──────────────────┘    └─────────────────┘
                             │
                             ▼
                   ┌──────────────────┐
                   │ Serverless Funcs │
                   │                  │
                   │ - AWS Lambda     │
                   │ - Vercel         │
                   │ - Netlify        │
                   │ - Custom APIs    │
                   └──────────────────┘
```

## Usage Examples

### 1. Client-Side SSE Connection

```javascript
const eventSource = new EventSource('http://localhost:8080/sse/client123');

eventSource.onopen = function(event) {
  console.log('Connected to SSE stream');
};

eventSource.addEventListener('connected', function(event) {
  const data = JSON.parse(event.data);
  console.log('Connection established:', data);
});

eventSource.addEventListener('function_response', function(event) {
  const response = JSON.parse(event.data);
  console.log('Function result:', response);
});

eventSource.addEventListener('heartbeat', function(event) {
  console.log('Heartbeat received');
});

eventSource.onerror = function(event) {
  console.error('SSE error:', event);
};
```

### 2. Register a Function

```bash
curl -X POST http://localhost:8080/admin/functions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "text-processor",
    "endpoint": "https://my-api.vercel.app/api/process",
    "method": "POST",
    "timeout": "30s",
    "description": "Processes text input"
  }'
```

### 3. Invoke a Function

```bash
curl -X POST http://localhost:8080/invoke/text-processor \
  -H "Content-Type: application/json" \
  -d '{
    "payload": {
      "text": "Hello, World!",
      "operation": "uppercase"
    },
    "client_id": "client123",
    "async": true
  }'
```

## Benefits

1. **Cost Optimization**: Serverless functions only run when needed
2. **Better UX**: No connection drops or reconnection delays  
3. **Scalability**: Independent scaling of connections vs compute
4. **Reliability**: Connection resilience separate from function health
5. **Real-time**: Instant delivery of function results via SSE
6. **Flexibility**: Support for any HTTP-based serverless function

## Monitoring

The system provides built-in monitoring and metrics:

- Connection count and client breakdown
- Function health and availability  
- Request/response metrics
- Redis connectivity status
- System uptime and performance

Access monitoring data via:
```bash
curl http://localhost:8080/admin/health
curl http://localhost:8080/admin/connections
```

## Development

**Build:**
```bash
go build -o virtualization-manager main.go
```

**Run tests:**
```bash
go test ./...
```

**Build Docker image:**
```bash
docker build -t sse-virtualization-manager .
```

## Contributing

This is an open-source project. Contributions are welcome!

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - feel free to use this in your projects!
