#!/usr/bin/env bash

set -e

# checks stuff and fails if no bueno

function main {
    go run cmd/rte-gen/*.go | diff rte_func.go -
    golint
}

main

