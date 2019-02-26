#!/usr/bin/env bash

set -e

# checks stuff and fails if no bueno

function main {
    local max_vars="${1}"
    local to_format=$(gofmt -l . | sort)
    if [ $(echo -n "$to_format" | wc -l) -gt 0 ]; then
        echo "files are misformatted: " >&2
        echo -n "$to_format" >&2
        exit 1
    fi

    go run ./internal/cmd/rte-gen -max-vars "${max_vars}" -output - | diff internal/funcs/funcs.go -
    go run ./internal/cmd/rte-gen -max-vars "${max_vars}" -test-output - | diff internal/funcs/funcs_test.go -

    go vet ./...

    golint -set_exit_status
}

main "${1}"
