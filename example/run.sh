#!/bin/bash

# Change to the root directory of the project
cd $(dirname $0)/..

# Run the server with the example configuration
go run go-json-server.go --config=./example/api.json
