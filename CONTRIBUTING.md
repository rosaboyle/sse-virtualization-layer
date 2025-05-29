# Contributing to SSE Virtualization Manager

We love your input! We want to make contributing to this project as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

## Pull Requests

Pull requests are the best way to propose changes to the codebase. We actively welcome your pull requests:

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Development Setup

### Prerequisites

- Go 1.24+
- Redis server
- Docker and Docker Compose (optional but recommended)

### Local Development

1. **Clone and setup:**
```bash
git clone https://github.com/yourusername/sse-virtualization-manager.git
cd sse-virtualization-manager
cp .env.example .env
```

2. **Install dependencies:**
```bash
make deps
```

3. **Start development environment:**
```bash
make docker-compose-up  # Starts Redis
make run               # In another terminal
```

4. **Run tests:**
```bash
make test
```

### Code Style

- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions focused and small
- Follow Go best practices and idioms

### Testing

- Write unit tests for new functionality
- Ensure existing tests pass
- Add integration tests for new endpoints
- Test both success and error scenarios

### Documentation

- Update README.md for user-facing changes
- Add/update code comments for complex logic
- Update API documentation for endpoint changes
- Include examples for new features

## Git Workflow

### Branching Strategy

- `main` - Production-ready code
- `develop` - Integration branch for features
- `feature/feature-name` - New features
- `bugfix/bug-description` - Bug fixes
- `hotfix/critical-fix` - Critical production fixes

### Commit Messages

Use clear and meaningful commit messages:

```
feat: add function timeout configuration
fix: resolve SSE connection drops on idle
docs: update API documentation
test: add unit tests for connection manager
refactor: improve error handling in gateway
```

### Branch Naming

- `feature/add-metrics-endpoint`
- `bugfix/fix-memory-leak`
- `docs/update-api-docs`
- `test/add-integration-tests`

## Issue Reporting

### Bug Reports

Great Bug Reports tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

### Feature Requests

Feature requests should include:

- Clear description of the feature
- Use case and motivation
- Detailed behavior description
- Any alternative solutions considered

## Architecture Guidelines

### Code Organization

```
pkg/
├── config/     # Configuration management
├── types/      # Shared data structures
├── redis/      # Redis client and operations
├── manager/    # Connection management
├── registry/   # Function registry
└── gateway/    # SSE gateway logic
```

### Design Principles

1. **Separation of Concerns**: Each package has a single responsibility
2. **Dependency Injection**: Use interfaces for testability
3. **Error Handling**: Explicit error handling with context
4. **Concurrency**: Use Go routines safely with proper synchronization
5. **Performance**: Optimize for high throughput and low latency

### API Design

- RESTful endpoints where appropriate
- Consistent naming conventions
- Proper HTTP status codes
- JSON request/response format
- Comprehensive error messages

## Testing Guidelines

### Unit Tests

```go
func TestConnectionManager_AddConnection(t *testing.T) {
    // Setup
    cm := NewConnectionManager(mockRedisClient)
    
    // Execute
    conn := cm.AddConnection("client1", "user1", nil)
    
    // Assert
    assert.NotNil(t, conn)
    assert.Equal(t, "client1", conn.ClientID)
}
```

### Integration Tests

```go
func TestSSEGateway_HandleConnection(t *testing.T) {
    // Setup test server
    server := httptest.NewServer(handler)
    defer server.Close()
    
    // Test SSE connection
    // ...
}
```

### Performance Tests

- Benchmark critical paths
- Test concurrent connections
- Measure memory usage
- Profile CPU usage

## Security Guidelines

- Validate all input data
- Use HTTPS in production
- Implement rate limiting
- Sanitize log output
- Follow OWASP guidelines

## Documentation

### Code Documentation

```go
// ConnectionManager manages SSE connections and their lifecycle.
// It provides thread-safe operations for adding, removing, and
// broadcasting messages to active connections.
type ConnectionManager struct {
    connections map[string]*types.Connection
    mutex       sync.RWMutex
    redisClient *redis.Client
}

// AddConnection creates a new SSE connection and stores it in Redis.
// Returns the created connection or error if storage fails.
func (cm *ConnectionManager) AddConnection(clientID, userID string, metadata map[string]string) (*types.Connection, error) {
    // Implementation...
}
```

### API Documentation

Document all endpoints with:
- Purpose and use case
- Request/response format
- Error conditions
- Example usage

## Performance Considerations

### Optimization Areas

1. **Connection Scaling**: Efficient connection management
2. **Memory Usage**: Minimize allocations in hot paths
3. **Redis Operations**: Batch operations where possible
4. **Goroutine Management**: Proper lifecycle management
5. **HTTP Performance**: Connection pooling and keep-alive

### Monitoring

- Add metrics for key operations
- Log performance-critical events
- Monitor memory and CPU usage
- Track connection lifecycle

## Release Process

### Version Numbering

We use [Semantic Versioning](http://semver.org/):
- MAJOR.MINOR.PATCH
- MAJOR: incompatible API changes
- MINOR: backwards-compatible functionality
- PATCH: backwards-compatible bug fixes

### Release Checklist

- [ ] All tests pass
- [ ] Documentation updated
- [ ] Changelog updated
- [ ] Version bumped
- [ ] Tagged release
- [ ] Docker image published

## Getting Help

- Create an issue for bugs or feature requests
- Join our discussions for questions
- Check existing documentation first
- Provide minimal reproducible examples

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (MIT License).

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes
- Project documentation

Thank you for contributing! 🚀
