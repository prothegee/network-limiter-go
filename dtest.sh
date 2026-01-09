#!/usr/bin/sh
set -e;

export CURRENT_DIR="$(pwd)";
export UNIT_TEST_DIR="$CURRENT_DIR/tests/unit_test";

cd $UNIT_TEST_DIR;
go test . -v;
echo "INFO: test in \"$UNIT_TEST_DIR\" finished";

cd $CURRENT_DIR;
