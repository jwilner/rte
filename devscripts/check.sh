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

    go run cmd/rte-gen/*.go -output - | diff rte_func.go -
    go run cmd/rte-gen/*.go -test-output - | diff rte_func_test.go -

    go vet ./...

    golint
}

main

