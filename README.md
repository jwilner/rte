# rte - routing table extraordinaire

[![Build Status](https://travis-ci.com/jwilner/rte.svg?branch=master)](https://travis-ci.com/jwilner/rte)
[![Go Report Card](https://goreportcard.com/badge/github.com/jwilner/rte)](https://goreportcard.com/report/github.com/jwilner/rte)

Dead simple, opinionated, performant routing.

- Only routes on method and path
- Routes are data to be manipulated and tested.
- Generate handler adapters means requests routed without any heap allocations
- Typed path parameters means you get to business logic faster

```go
package main

import (
    "fmt"
    "github.com/jwilner/rte"
    "net/http"
)

func main() {
    http.Handle("/", rte.Must(
        rte.FuncS1I2(
            "GET", "/foo/:foo_name/bar/:bar_id/baz/:baz_id/",
            func(w http.ResponseWriter, r *http.Request, fooName string, barID, bazID int64) {
                _, _ = fmt.Fprintf(w, "fooID: %v, barID: %v, bazID: %v\n", fooName, barID, bazID)
            },
        ),
        rte.Func(
            "POST", "/foo",
            func(w http.ResponseWriter, _ *http.Request) {
                _, _ = w.Write([]byte("handled by foo"))
            },
        ),
    ))
}
```

## Development

Check out the [Makefile](Makefile) for dev entrypoints.

TLDR:
- `make test`
- `make gen` (builds gen binary and regenerates code)
- `make lint` (runs `golint` -- install with `go get -u golang.org/x/lint/golint`)

## CI

Travis builds. In addition to tests, the build is gated on `golint` and whether the checked-in generated code matches the code as currently generated.
