# rte

[![Build Status](https://travis-ci.com/jwilner/rte.svg?branch=master)](https://travis-ci.com/jwilner/rte)
[![Go Report Card](https://goreportcard.com/badge/github.com/jwilner/rte)](https://goreportcard.com/report/github.com/jwilner/rte)

Dead simple, opinionated, performant routing.

- Only routes on method and path
- Routes are data to be manipulated and tested.

```go
package main

import (
    "github.com/jwilner/rte"
    "net/http"
    "strings"
)

func main() {
    rtes := []rte.Route{
        rte.Func("GET", "/foo/*/bar/*", func(w http.ResponseWriter, r *http.Request) {
            params := rte.PathVars(r)
            _, _ = w.Write([]byte(strings.Join(params, "-")))
        }),
        rte.Func("POST", "/foo", func(w http.ResponseWriter, _ *http.Request) {
            _, _ = w.Write([]byte("handled by foo"))
        }),
    }

    tbl := rte.Must(rtes...)

    http.Handle("/", tbl)
}
```
