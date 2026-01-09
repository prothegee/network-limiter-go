#!/usr/bin/bash
set -e;

export PWD="$(pwd)";
export PROTO_DIR="$PWD/protobuf";

# location.proto
protoc "$PROTO_DIR/location.proto" \
    --go_out="$PWD" \
    --go-grpc_out="$PWD" \
    --proto_path="$PROTO_DIR";

