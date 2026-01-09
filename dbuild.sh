#!/usr/bin/sh
set -e;

mkdir -p bin;

export TARGET_DIR="$(pwd)/bin";

echo "NOTE: all build goes to $TARGET_DIR";

export SERVER_GRPC_SOURCE="$(pwd)/cmd/server_grpc";
export SERVER_GRPC_TARGET="$TARGET_DIR/server_grpc/main";
export SERVER_NETHTTP_SOURCE="$(pwd)/cmd/server_nethttp";
export SERVER_NETHTTP_TARGET="$TARGET_DIR/server_nethttp/main";

echo "building: $SERVER_GRPC_SOURCE";
echo "- target: $SERVER_GRPC_TARGET"
go build -o $SERVER_GRPC_TARGET $SERVER_GRPC_SOURCE;

echo "building: $SERVER_NETHTTP_SOURCE";
echo "- target: $SERVER_NETHTTP_TARGET";
go build -o $SERVER_NETHTTP_TARGET $SERVER_NETHTTP_SOURCE;

