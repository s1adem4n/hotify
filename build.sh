#!/bin/bash

# get current commit hash
COMMIT_HASH=$(git rev-parse HEAD)

# first arg is the output file
if [ -z "$1" ]; then
  echo "Usage: $0 <output_file>"
  exit 1
fi

cd webui
bun install
bun run build
cd ..

# build the server and inject the commit hash
go build -ldflags "-X main.CommitHash=$COMMIT_HASH" -o $1 server/main.go
