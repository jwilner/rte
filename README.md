# rte - routing table extraordinaire

[![Build Status](https://travis-ci.com/jwilner/rte.svg?branch=master)](https://travis-ci.com/jwilner/rte)
[![Go Report Card](https://goreportcard.com/badge/github.com/jwilner/rte)](https://goreportcard.com/report/github.com/jwilner/rte)
[![GoDoc](https://godoc.org/github.com/jwilner/rte?status.svg)](https://godoc.org/github.com/jwilner/rte)
[![Coverage Status](https://coveralls.io/repos/github/jwilner/rte/badge.svg?branch=coverage)](https://coveralls.io/github/jwilner/rte?branch=coverage)

Dead simple, opinionated, performant routing.

- Intuitive, legible interface; encourages treating routing configuration as data to be passed around and manipulated.
- Extracted path variables are matched to type signature; handlers get to business logic faster, code is more explicit, and programming errors are surfaced before request handling time.
- Fast AF -- completely avoids the heap during request handling; markedly faster than any other router in the [go-http-routing-benchmark suite](#performance).

```go
package main

import (
    "fmt"
    "github.com/jwilner/rte"
    "log"
    "net/http"
)

func main() {
    log.Fatal(http.ListenAndServe(":8080", rte.Must(rte.Routes(
        "/foo", rte.Routes(
            "POST", func(w http.ResponseWriter, r *http.Request) {
                // POST /foo
            },
            "/:id", rte.Routes(
                "GET", func(w http.ResponseWriter, r *http.Request, id string) {
                    // GET /foo/:id
                },
                "PUT", func(w http.ResponseWriter, r *http.Request, id string) {
                    // PUT /foo/:id
                },
                "DELETE", func(w http.ResponseWriter, r *http.Request, id string) {
                    // DELETE /foo/:id
                },
                rte.MethodAny, func(w http.ResponseWriter, r *http.Request, id string) {
                    // serve a 405
                },
                "POST /bar", func(w http.ResponseWriter, r *http.Request, id string) {
                    // POST /foo/:id/bar
                },
            ),
        ),
    ))))
}
```
## Usage

`rte.Route` values are passed to `rte.Must` or `rte.New`, which constructs an `*rte.Table`. There are plenty of examples here, but also check out the [go docs](https://godoc.org/github.com/jwilner/rte#pkg-examples).

### rte.Route

The core value is the simple `rte.Route` struct:
```go
route := rte.Route{Method: "GET", Path: "/health", Handler: healthHandler}
```
You can construct and manipulate them as you would any struct; the handler can be a standard `http.Handler`, `http.HandlerFunc`, or one of:
```go
func(http.ResponseWriter, *http.Request, string)
func(http.ResponseWriter, *http.Request, string, string)
func(http.ResponseWriter, *http.Request, string, string, string)
func(http.ResponseWriter, *http.Request, [N]string) // where N is a number between 1 and 8
```
If the handler has string parameters, RTE injects any variables indicted w/in the path into function signature. For signatures of 4 or more, only array signatures are provided; arrays, rather than slices, are used to avoid heap allocations -- and to be explicit. **It's a configuration error** if the number of path variables doesn't match the number of function parameters (an exception is made for zero string parameter functions -- they can be used with any number of path variables).

Each struct can also be assigned middleware behavior:
```go
route.Middleware = func(w http.ResponseWriter, r *http.Request, next http.Handler) {
    _, _ = fmt.Println(w, "Before request")
    next.ServeHttp(w, r)
    _, _ = fmt.Println(w, "After request")
}
```

### rte.Routes constructor

When it's useful, there's an overloaded variadic constructor:
```go
routes := rte.Routes(
    "GET /health", healthHandler,
    "GET /foo/:foo_id", func(w http.ResponseWriter, r *http.Request, fooID string) {
        _, _ = fmt.Fprintf(w, "fooID: %v", fooID)
    },
)
```

It can be used to construct hierarchical routes:
```go
routes := rte.Routes(
    "/foo", rte.Routes(
        "POST", func(w http.ResponseWriter, r *http.Request) {
            // POST /foo
        },
        "/:foo_id", rte.Routes(
            "GET", func(w http.ResponseWriter, r *http.Request, fooID string) {
                // GET /foo/:foo_id
            },
            "PUT", func(w http.ResponseWriter, r *http.Request, fooID string) {
                // PUT /foo/:foo_id
            },
        ), myMiddleware
    )
)
```
The above is exactly equivalent to:
```go
routes := []rte.Route {
    {
        Method: "POST", Path: "/foo",
        Handler: func(http.ResponseWriter, *http.Request) {
            // POST /foo
        },
    },
    {
        Method: "GET", Path: "/foo/:foo_id",
        Handler: func(http.ResponseWriter, *http.Request, id string) {
            // GET /foo/:foo_id
        },
        Middleware: myMiddleware
    },
    {
        Method: "PUT", Path: "/foo/:foo_id",
        Handler: func(w http.ResponseWriter, r *http.Request, id string) {
            // PUT /foo/:foo_id
        },
        Middleware: myMiddleware
    },
}
```

See [examples_test.go](examples_test.go) or [go docs](https://godoc.org/github.com/jwilner/rte#example-Routes) for more examples.

### Compiling the routing table

Zero or more routes are combined to create a `Table` handler using either `New` or `Must`:
```go
var routes []rte.Route
tbl, err := rte.New(routes) // errors on misconfiguration
tbl = rte.Must(routes) // panics on misconfiguration
```
If you're dynamically constructing your routes, the returned `rte.Error` type helps you recover from misconfiguration.

`*rte.Table` satisfies the standard `http.Handler` interface and can be used with standard Go http utilities.

### Extras

RTE provides a few basic values and functions to help with common patterns. Many of these functions take in a `[]Route` and return a new, potentially modified `[]Route`, in keeping with the [design principles](#design-principles).

#### MethodAny

RTE performs wildcard matching in paths with the `:` syntax; it can also perform wildcard matching of methods via the use of `rte.MethodAny`. You can use `rte.MethodAny` anywhere you would a normal HTTP method; it will match any requests to the path that don't match an explicit, static method:

```go
rte.Routes(
    // handles GETs to /
    "GET /", func(http.ResponseWriter, *http.Request){},
    // handles POST PUT, OPTIONS, etc. to /
    rte.MethodAny+" /", func(http.ResponseWriter, *http.Request){},
)
```

#### DefaultMethod

`rte.DefaultMethod` adds a `rte.MethodAny` handler to every path; useful if you want to serve 405s for all routes.

```go
reflect.DeepEqual(
    rte.DefaultMethod(
        hndlr405,
        []rte.Route {
            {Method: "GET", Path: "/foo", Handler: fooHandler},
            {Method: "POST", Path: "/foo", Handler: postFooHandler},
            {Method: "GET", Path: "/bar", Handler: barHandler},
        },
    ),
    []rte.Route {
        {Method: "GET", Path: "/foo", Handler: fooHandler},
        {Method: rte.MethodAny, Path: "/foo", Handler: hndlr405},
        {Method: "POST", Path: "/foo", Handler: postFooHandler},
        {Method: "GET", Path: "/bar", Handler: barHandler},
        {Method: rte.MethodAny, Path: "/bar", Handler: hnldr405},
    },
)
```

#### Wrap

`rte.Wrap` adds middleware behavior to every contained path; if a middleware is already set, the new middleware will be wrapped around it -- so that the stack will have the new middleware at the top, the old middleware in the middle, and the handler at the bottom.

```go
reflect.DeepEqual(
    rte.Wrap(
        myMiddleware,
        []rte.Route {
            {Method: "GET", Path: "/foo", Handler: myHandler},
        },
    ),
    []rte.Route {
        {Method: "GET", Path: "/foo", Handler: myHandler, Middleware: myMiddleware},
    },
)
```

#### OptTrailingSlash

OptTrailingSlash makes each handler also match its slashed or not-slashed version.

```go
reflect.DeepEqual(
    rte.OptTrailingSlash(
        []rte.Route {
            {Method: "GET", Path: "/foo", Handler: myHandler},
            {Method: "POST", Path: "/bar/", Handler: barHandler},
        },
    ),
    []rte.Route {
        {Method: "GET", Path: "/foo", Handler: myHandler},
        {Method: "GET", Path: "/foo/", Handler: myHandler},
        {Method: "POST", Path: "/bar/", Handler: barHandler},
        {Method: "POST", Path: "/bar", Handler: barHandler},
    },
)
```

#### more

Check out the [go docs](https://godoc.org/github.com/jwilner/rte) for still more extras.


## Trade-offs

It's important to note that RTE uses a fixed size array of strings for path variables paired with generated code to avoid heap allocations; currently, this number is fixed at 8, which means that **RTE does not support routes with more than eight path variables** (doing so will cause an error or panic). The author is deeply skeptical that anyone actually really needs more than eight path variables; that said, it's a design goal to provide support higher numbers once the right packaging technique is found.

## Performance

Modern Go routers place a lot of emphasis on speed. There's plenty of room for skepticism about this attitude, as most web application will be IO bound. Nonetheless, this is the barrier for entry to the space these days. To this end, RTE completely avoids performing zero heap allocations while serving requests and uses a relatively optimized data structure (a compressed trie) to route requests.

Numbers are drawn from this [fork](https://github.com/jwilner/go-http-routing-benchmark) of [go-http-routing-benchmark](https://github.com/julienschmidt/go-http-routing-benchmark) (which appears unmaintained). The numbers below are from a 2013 MB Pro with a 2.6 GHz i5 and 8 GB ram.

|Single Param Micro Benchmark| | | | |
|---|---|---|---|---|
|**RTE**|20000000|69.1 ns/op|0 B/op|0 allocs/op|
|Gin|20000000|86.0 ns/op|0 B/op|0 allocs/op|
|LARS|20000000|90.0 ns/op|0 B/op|0 allocs/op|
|Echo|20000000|107 ns/op|0 B/op|0 allocs/op|
|HttpRouter|10000000|139 ns/op|32 B/op|1 allocs/op

|Google Plus API with 1 Param | | | | |
|---|---|---|---|---|
|**RTE**|20000000|101 ns/op|0 B/op|0 allocs/op|
|LARS|10000000|116 ns/op|0 B/op|0 allocs/op|
|Gin|20000000|123 ns/op|0 B/op|0 allocs/op|
|Echo|10000000|153 ns/op|0 B/op|0 allocs/op|
|HttpRouter|10000000|239 ns/op|64 B/op|1 allocs/op|

|Five Param Micro Benchmark | | | | |
|---|---|---|---|---|
|**RTE**|10000000|119 ns/op|0 B/op|0 allocs/op|
|LARS|10000000|153 ns/op|0 B/op|0 allocs/op|
|Gin|10000000|179 ns/op|0 B/op|0 allocs/op|
|Echo|5000000|319 ns/op|0 B/op|0 allocs/op|
|HttpRouter|3000000|445 ns/op|160 B/op|1 allocs/op|

|Google Plus API with 2 Params| | | | |
|---|---|---|---|---|
|LARS|10000000|149 ns/op|0 B/op|0 allocs/op|
|**RTE**|10000000|153 ns/op|0 B/op|0 allocs/op|
|Gin|10000000|164 ns/op|0 B/op|0 allocs/op|
|Echo|10000000|252 ns/op|0 B/op|0 allocs/op|
|HttpRouter|5000000|271 ns/op|64 B/op|1 allocs/op|

|Github API Single Param | | | | |
|---|---|---|---|---|
|**RTE**|10000000|152 ns/op|0 B/op|0 allocs/op|
|LARS|10000000|181 ns/op|0 B/op|0 allocs/op|
|Gin|10000000|192 ns/op|0 B/op|0 allocs/op|
|Echo|5000000|307 ns/op|0 B/op|0 allocs/op|
|HttpRouter|5000000|337 ns/op|96 B/op|1 allocs/op|

|Github API All | | | | |
|---|---|---|---|---|
|**RTE**|50000|32830 ns/op|0 B/op|0 allocs/op|
|LARS|50000|34791 ns/op|0 B/op|0 allocs/op|
|Gin|50000|38353 ns/op|0 B/op|0 allocs/op|
|HttpRouter|20000|58992 ns/op|13792 B/op|167 allocs/op|
|Echo|20000|91093 ns/op|0 B/op|0 allocs/op|

## Design principles

### simplicity

RTE attempts to follow Golang's lead by avoiding features which complicate routing behavior when a viable alternative is available to the user. For example, RTE chooses not to allow users to specify path variables with types -- e.g. `{foo_id:int}` -- or catch-all paths -- e.g. a `/foo/*` matching `/foo/bar/blah`. Supporting either of those features would introduce complicated precedence behavior, and simple alternatives exist for users.

Additionally, RTE aims to have defined behavior in all circumstances and to document and prove that behavior with unit tests.

### limiting state space

Many modern routers coordinate by mutating a common data structure -- unsurprisingly, usually called a `Router`. In larger applications that pass around the routers and subrouters, setting different flags and options in different locations, the state of the router can be at best hard to reason about and at worst undefined -- it is not uncommon for certain feature combinations to fail or have unexpected (or insecure) results.

By centering on the simple `rte.Route` and not exposing any mutable state, RTE keeps its interface simple to understand, while also simplifying its own internals . Because most routing libraries focus on the mutable router object, they do not have explicit finalization, and thus their internal logic must remain open to modification at any time -- RTE does not have this problem, and benefits.

When a routing feature is necessary, it will usually be added as an helper method orthogonal to the rest of the API. For example, rather than providing a method `OptTrailingSlash(enabled bool)` on `*rte.Table` and pushing complexity into the routing logic, RTE provides the pure function `rte.OptTrailingSlash(routes []rte.Route) []rte.Route`, which adds the new `rte.Route`s necessary to optionally match trailing slashes, while the routing logic remains unchanged.

```go
reflect.DeepEqual(
    rte.OptTrailingSlash(
        []Route{
            {Method: "GET", Path: "/foo", Handler: myHandler},
        },
    ),
    []Route {
        {Method: "GET", Path: "/foo", Handler: myHandler},
        {Method: "GET", Path: "/foo/", Handler: myHandler},
    }
)
```

## Development

Check out the [Makefile](Makefile) for dev entrypoints.

TLDR:
- `make test`
- `make test-cover`
- `make gen` (regenerates internal code)
- `make check` (requires `golint` -- install with `go get -u golang.org/x/lint/golint`)

## CI

Travis builds. In addition to tests, the build is gated on `golint` and whether the checked-in generated code matches the code as currently generated.
