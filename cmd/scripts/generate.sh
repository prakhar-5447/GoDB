#!/bin/bash

# Define paths
PROTO_DIR="./internal/proto"  # Relative path from /cmd/scripts/
OUT_DIR="./internal"         # Where to store generated gRPC files

# Create output directory if it doesn't exist
mkdir -p "$OUT_DIR"

# Generate gRPC files
protoc \
  --proto_path="$PROTO_DIR" \
  --go_out="$OUT_DIR" \
  --go-grpc_out="$OUT_DIR" \
  "$PROTO_DIR"/*.proto

echo "✅ gRPC files generated successfully in $OUT_DIR"
