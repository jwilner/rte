#!/usr/bin/env bash

set -e

# checks stuff and fails if no bueno

function main {
    local to_format=$(gofmt -l . | sort)
    if [ $(echo -n "$to_format" | wc -l) -gt 0 ]; then
        echo "files are misformatted: " >&2
        echo -n "$to_format" >&2
        exit 1
    fi
    go run cmd/rte-gen/*.go | diff rte_func.go -
    golint
}

main

