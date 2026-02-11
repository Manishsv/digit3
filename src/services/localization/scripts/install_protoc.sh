#!/bin/bash

# Install protoc compiler
sudo apt-get update
sudo apt-get install -y protobuf-compiler

# Install specific versions of Go plugins for protoc
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Add Go bin to PATH if not already there
export PATH="$PATH:$(go env GOPATH)/bin" 