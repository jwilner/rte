# rte - routing table extraordinaire

[![Build Status](https://travis-ci.com/jwilner/rte.svg?branch=master)](https://travis-ci.com/jwilner/rte)
[![Go Report Card](https://goreportcard.com/badge/github.com/jwilner/rte)](https://goreportcard.com/report/github.com/jwilner/rte)
[![GoDoc](https://godoc.org/github.com/jwilner/rte?status.svg)](https://godoc.org/github.com/jwilner/rte)
[![Coverage Status](https://coveralls.io/repos/github/jwilner/rte/badge.svg?branch=coverage)](https://coveralls.io/github/jwilner/rte?branch=coverage)

Dead simple, opinionated, performant routing.

- Only routes on method and path
- Routes are data to be manipulated and tested.
- Generated handler adapters means requests routed without any heap allocations

```go
package main

import (
    "fmt"
    "github.com/jwilner/rte"
    "net/http"
)

func main() {
    http.Handle("/", rte.Must(rte.Routes(
        "/my-resource", rte.Routes(
            "POST", func(w http.ResponseWriter, r *http.Request) {
                // create
            },
            "/:id", rte.Routes(
                "GET", func(w http.ResponseWriter, r *http.Request, id string) {
                    // read
                },
                "PUT", func(w http.ResponseWriter, r *http.Request, id string) {
                    // update
                },
                "DELETE", func(w http.ResponseWriter, r *http.Request, id string) {
                    // delete
                },
                rte.MethodAll, func(w http.ResponseWriter, r *http.Request, id string) {
                    // serve a 405
                },
            ),
        ),
    )))
}
```

## Development

Check out the [Makefile](Makefile) for dev entrypoints.

TLDR:
- `make test`
- `make gen` (regenerates internal code)
- `make check` (requires `golint` -- install with `go get -u golang.org/x/lint/golint`)

## CI

Travis builds. In addition to tests, the build is gated on `golint` and whether the checked-in generated code matches the code as currently generated.
