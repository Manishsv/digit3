#!/bin/bash

# Create the output directory if it doesn't exist
mkdir -p api/proto/localization/v1

# Generate the gRPC code
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    api/proto/localization/v1/localization.proto 