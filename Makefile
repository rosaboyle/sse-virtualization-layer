BINARY_NAME=virtualization-manager
DOCKER_IMAGE=sse-virtualization-layer

.PHONY: build run test clean docker-build docker-run docker-compose-up docker-compose-down deps

# Build the application
build:
	go build -o $(BINARY_NAME) main.go

# Run the application locally
run:
	go run main.go

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE) .

# Run Docker container
docker-run:
	docker run -p 8080:8080 -e REDIS_ADDR=host.docker.internal:6379 $(DOCKER_IMAGE)

# Start with Docker Compose
docker-compose-up:
	docker-compose up -d

# Stop Docker Compose
docker-compose-down:
	docker-compose down

# View Docker Compose logs
logs:
	docker-compose logs -f

# Redis CLI (requires Redis container to be running)
redis-cli:
	docker exec -it $$(docker-compose ps -q redis) redis-cli

# Check health
health:
	curl -f http://localhost:8080/admin/health

# Register example function
register-example:
	curl -X POST http://localhost:8080/admin/functions \
		-H "Content-Type: application/json" \
		-d '{"name": "echo", "endpoint": "https://httpbin.org/post", "method": "POST", "description": "Echo service"}'

# Test SSE connection
test-sse:
	curl -N -H "Accept: text/event-stream" "http://localhost:8080/sse/test-client?app=test"

# Invoke example function
invoke-example:
	curl -X POST http://localhost:8080/invoke/echo \
		-H "Content-Type: application/json" \
		-d '{"payload": {"message": "Hello World"}, "client_id": "test-client"}'

# Development workflow
dev: deps build run

# Production deployment
deploy: docker-build docker-compose-up

# Full cleanup
nuke: docker-compose-down clean
	docker rmi $(DOCKER_IMAGE) || true
	docker volume prune -f
