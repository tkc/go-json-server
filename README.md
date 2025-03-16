
![Go](https://github.com/tkc/go-json-server/workflows/Go/badge.svg?branch=master)

```
  ___          _                 ___                      
 / __|___   _ | |__ _ ___ _ _   / __| ___ _ ___ _____ _ _ 
| (_ / _ \ | || / _` / _ \ ' \  \__ \/ -_) '_\ V / -_) '_|
 \___\___/  \__/\__,_\___/_||_| |___/\___|_|  \_/\___|_|  
                                                          
```                                                

# Go JSON Server

A powerful, flexible and efficient mock server for testing and development environments. 
Serve static JSON APIs and files with customizable routes and responses.

## Requirements

- Go 1.22.3 or later

## Key Features

- ✅ **Configuration-driven API** - Define your endpoints in a simple JSON file
- ✅ **Hot-reloading** - Changes to configuration are detected and applied without server restart
- ✅ **Response caching** - Improved performance with configurable TTL
- ✅ **Path parameters** - Support for dynamic route parameters like `/users/:id`
- ✅ **Static file server** - Serve files from specified directories
- ✅ **Middleware architecture** - Logging, CORS, timeout, and panic recovery included
- ✅ **Structured logging** - Configurable log levels with JSON or text formats
- ✅ **Request ID tracking** - Assign unique IDs to each request for better traceability
- ✅ **Error handling** - Detailed error responses and validation
- ✅ **Command-line flags** - Override configuration settings via command line arguments

## Installation

```bash
# Install the latest version
go install github.com/tkc/go-json-server@latest

# Or using go get 
go get -u github.com/tkc/go-json-server
```

## Getting Started

### 1. Create your API configuration file

Create a file named `api.json` in your project directory:

```json
{
  "port": 3000,
  "logLevel": "info",
  "logFormat": "text",
  "endpoints": [
    {
      "method": "GET",
      "status": 200,
      "path": "/",
      "jsonPath": "./health-check.json"
    },
    {
      "method": "GET",
      "status": 200,
      "path": "/users",
      "jsonPath": "./users.json"
    },
    {
      "method": "GET",
      "status": 200,
      "path": "/user/:id",
      "jsonPath": "./user.json"
    },
    {
      "path": "/files",
      "folder": "./static"
    }
  ]
}
```

### 2. Create your JSON response files

Create the JSON files referenced in your configuration:

**health-check.json**
```json
{
  "status": "ok",
  "message": "go-json-server running",
  "version": "1.0.0"
}
```

**users.json**
```json
[
  {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com"
  },
  {
    "id": 2,
    "name": "Jane Smith",
    "email": "jane@example.com"
  }
]
```

**user.json**
```json
{
  "id": ":id",
  "name": "John Doe",
  "email": "john@example.com",
  "address": "123 Main St"
}
```

### 3. Start your server

```bash
go-json-server
```

Or with custom configuration:

```bash
go-json-server --config=./custom-config.json --port=8080
```

## Running, Building and Testing

### Running the Application

You can run the application directly using Go:

```bash
# Run from source code
go run go-json-server.go

# Run with custom configuration
go run go-json-server.go --config=./example/api.json --port=8080 --log-level=debug
```

### Building the Application

Build a binary for your current platform:

```bash
# Simple build
go build -o go-json-server

# Build with version information
go build -ldflags="-X 'main.Version=v1.0.0'" -o go-json-server
```

Build for multiple platforms:

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o go-json-server-linux-amd64

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o go-json-server-darwin-amd64

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o go-json-server-windows-amd64.exe
```

### Testing

Run all tests:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test -v ./src/config
go test -v ./src/handler
go test -v ./src/logger
go test -v ./src/middleware
```

Generate and view test coverage:

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Get coverage percentage
go tool cover -func=coverage.out
```

Run benchmark tests:

```bash
# Run benchmark tests
go test -bench=. ./...

# Run benchmark with memory allocation stats
go test -bench=. -benchmem ./...
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `port` | Server port | 3000 |
| `host` | Server host | "" (all interfaces) |
| `logLevel` | Logging level (debug, info, warn, error, fatal) | "info" |
| `logFormat` | Log format (text, json) | "text" |
| `logPath` | Path to log file (stdout, stderr, or file path) | "stdout" |
| `endpoints` | Array of endpoint configurations | [] |

### Endpoint Configuration

| Option | Description | Required |
|--------|-------------|----------|
| `method` | HTTP method (GET, POST, PUT, DELETE, etc.) | Yes (for API endpoints) |
| `status` | HTTP response status code | Yes (for API endpoints) |
| `path` | URL path for the endpoint | Yes |
| `jsonPath` | Path to JSON response file | Yes (for API endpoints) |
| `folder` | Path to static files directory | Yes (for file server endpoints) |

## Path Parameters

You can use path parameters in your routes by prefixing a path segment with a colon:

```json
{
  "method": "GET",
  "status": 200,
  "path": "/users/:id/posts/:postId",
  "jsonPath": "./user-post.json"
}
```

The parameter values will be available in the JSON response by using the same parameter name with a colon:

```json
{
  "userId": ":id",
  "postId": ":postId",
  "title": "Sample Post"
}
```

## Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to configuration file | "./api.json" |
| `--port` | Override server port from config | Config port value |
| `--log-level` | Override log level from config | Config log level |
| `--log-format` | Override log format from config | Config log format |
| `--log-path` | Override log path from config | Config log path |
| `--cache-ttl` | Cache TTL in seconds | 300 (5 minutes) |

## Development Workflow

1. **Clone the repository**:
   ```bash
   git clone https://github.com/tkc/go-json-server.git
   cd go-json-server
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Make your changes**:
   - Add features
   - Fix bugs
   - Update documentation

4. **Run tests**:
   ```bash
   go test ./...
   ```

5. **Build and run**:
   ```bash
   go build -o go-json-server
   ./go-json-server --config=./example/api.json
   ```

## Docker Support

You can run go-json-server in a Docker container:

```dockerfile
FROM golang:1.22.3-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /go-json-server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /go-json-server /usr/local/bin/
COPY api.json .
COPY *.json .
EXPOSE 3000
CMD ["go-json-server"]
```

Build and run with Docker:

```bash
# Build Docker image
docker build -t go-json-server .

# Run Docker container
docker run -p 3000:3000 -v $(pwd)/example:/app/example go-json-server --config=/app/example/api.json
```

## Advanced Examples

### Authentication Middleware Example

To add basic authentication to your API:

```go
// In your main.go custom implementation
auth := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user, pass, ok := r.BasicAuth()
        if !ok || user != "admin" || pass != "secret" {
            w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte(`{"error": "unauthorized"}`))
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Apply middleware
handler := middleware.Chain(
    middleware.Logger(logger),
    middleware.CORS(),
    middleware.Recovery(logger),
    middleware.RequestID(),
    auth,
)(server.HandleRequest)
```

## Roadmap

- [ ] GraphQL support
- [ ] WebSocket support
- [ ] JWT authentication
- [ ] Response delay simulation
- [ ] Integration with Swagger/OpenAPI
- [ ] Proxy mode
- [ ] Request validation 
- [ ] Response templating
- [ ] Interactive web UI for API exploration

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT ✨
