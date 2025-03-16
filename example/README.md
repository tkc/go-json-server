# Go JSON Server Example

This directory contains example files for the Go JSON Server. These examples demonstrate various features of the server including:

- Basic API endpoints
- Path parameters (`:id`, `:userId`, `:postId`)
- Static file serving
- Different HTTP methods (GET, POST)
- Various response status codes

## Files Overview

- `api.json` - The main configuration file for the server
- `health-check.json` - Simple health check endpoint response
- `users.json` - List of users
- `user-detail.json` - Detailed user information with path parameter support
- `posts.json` - List of blog posts
- `post-detail.json` - Detailed post information with path parameter support
- `user-post.json` - Demonstrates multiple path parameters in one endpoint
- `user-created.json` - Example response for a POST request
- `static/` - Directory for static files
  - `sample.jpg` - Example image file
  - `index.html` - Example HTML documentation page

## Running the Example

From the root directory of the project:

```bash
go run go-json-server.go --config=./example/api.json
```

Or if you've installed the binary:

```bash
go-json-server --config=./example/api.json
```

## Testing the Endpoints

You can use curl, Postman, or any HTTP client to test the endpoints:

```bash
# Get health check
curl http://localhost:3000/

# Get all users
curl http://localhost:3000/users

# Get user with ID 1
curl http://localhost:3000/user/1

# Get all posts
curl http://localhost:3000/posts

# Get post with ID 2
curl http://localhost:3000/posts/2

# Get post 3 by user 2
curl http://localhost:3000/users/2/posts/3

# Create a new user
curl -X POST http://localhost:3000/users

# Access static HTML page
curl http://localhost:3000/static/index.html
# Or open in browser: http://localhost:3000/static/index.html
```

## Customizing

Feel free to modify these example files or create your own to experiment with the Go JSON Server's capabilities.
