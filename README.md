# rte - routing table extraordinaire

[![Tests](https://github.com/jwilner/rte/workflows/tests/badge.svg)](https://github.com/jwilner/rte/actions?query=workflow%3Atests+branch%3Amain)
[![Lint](https://github.com/jwilner/rte/workflows/lint/badge.svg)](https://github.com/jwilner/rte/actions?query=workflow%3Alint+branch%3Amain)
[![GoDoc](https://godoc.org/github.com/jwilner/rte?status.svg)](https://godoc.org/github.com/jwilner/rte)
[![Coverage](https://coveralls.io/repos/github/jwilner/rte/badge.svg?branch=coverage)](https://coveralls.io/github/jwilner/rte?branch=coverage)

Simple, opinionated, performant routing.

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
                    // Match any other HTTP method on /foo/:id (e.g. to serve a 405)
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

It's important to note that RTE uses a fixed size array of strings for path variables paired with generated code to avoid heap allocations; currently, this number is fixed at 8, which means that **RTE does not support routes with more than eight path variables** (doing so will cause an error or panic). The author is deeply skeptical that anyone actually really needs more than eight path variables; that said, it's a design goal to provide support for higher numbers once the right packaging technique is found.

## Performance

Modern Go routers place a lot of emphasis on speed. There's plenty of room for skepticism about this attitude, as most web application will be IO bound. Nonetheless, this is the barrier for entry to the space these days. To this end, RTE completely avoids performing zero heap allocations while serving requests and uses a relatively optimized data structure (a compressed trie) to route requests.

Benchmarks are drawn from this [fork](https://github.com/jwilner/go-http-routing-benchmark) of [go-http-routing-benchmark](https://github.com/julienschmidt/go-http-routing-benchmark) (which appears unmaintained). The benchmarks were run on a 2013 MB Pro with a 2.6 GHz i5 and 8 GB ram. For fun, RTE is compared to some of the most popular Go router layers below.

| Single Param Micro Benchmark| Reps | ns/op | B/op | allocs/op |
|---|---|---|---|---|
| **RTE** | 20000000 | 64.0 | 0 | 0 |
| Gin | 20000000 | 79.4 | 0 | 0 |
| Echo | 20000000 | 105 | 0 | 0 |
| HttpRouter | 10000000 | 138 | 32 | 1 |
| Beego | 1000000 | 1744 | 352 | 3 |
| GorillaMux | 300000 | 3775 | 1280 | 10 |
| Martini | 200000 | 6891 | 1072 | 10 |

| Five Param Micro Benchmark | Reps | ns/op | B/op | allocs/op |
|---|---|---|---|---|
| **RTE** | 20000000 | 116 | 0 | 0 |
| Gin | 10000000 | 137 | 0 | 0 |
| Echo | 5000000 | 253 | 0 | 0 |
| HttpRouter | 3000000 | 416 | 160 | 1 |
| Beego | 1000000 | 2036 | 352 | 3 |
| GorillaMux | 300000 | 5194 | 1344 | 10 |
| Martini | 200000 | 8091 | 1232 | 11 |

| Github API with 1 Param | Reps | ns/op | B/op | allocs/op |
|---|---|---|---|---|
| **RTE** | 10000000 | 156 | 0 | 0 |
| Gin | 10000000 | 184 | 0 | 0 |
| Echo | 5000000 | 266 | 0 | 0 |
| HttpRouter | 5000000 | 296 | 96 | 1 |
| Beego | 1000000 | 2018 | 352 | 3 |
| GorillaMux | 200000 | 10949 | 1296 | 10 |
| Martini | 100000 | 14957 | 1152 | 11 |

| Google Plus API with 1 Param | Reps | ns/op | B/op | allocs/op |
|---|---|---|---|---|
| **RTE** | 20000000 | 89.1 | 0 | 0 |
| Gin | 20000000 | 123 | 0 | 0 |
| Echo | 10000000 | 143 | 0 | 0 |
| HttpRouter | 10000000 | 185 | 64 | 1 |
| Beego | 1000000 | 1488 | 352 | 3 |
| GorillaMux | 300000 | 4053 | 1280 | 10 |
| Martini | 200000 | 6375 | 1072 | 10 |

| Google Plus API with 2 Params | Reps | ns/op | B/op | allocs/op |
|---|---|---|---|---|
| **RTE** | 10000000 | 139 | 0 | 0 |
| Gin | 10000000 | 141 | 0 | 0 |
| HttpRouter | 5000000 | 225 | 64 | 1 |
| Echo | 10000000 | 237 | 0 | 0 |
| Beego | 1000000 | 1646 | 352 | 3 |
| GorillaMux | 200000 | 8675 | 1296 | 10 |
| Martini | 100000 | 13586 | 1200 | 13 |

## Design principles

### simplicity

RTE attempts to follow Golang's lead by avoiding features which complicate routing behavior when a viable alternative is available to the user. For example, RTE chooses not to allow users to specify path variables with types -- e.g. `{foo_id:int}` -- or catch-all paths -- e.g. a `/foo/*` matching `/foo/bar/blah`. Supporting either of those features would introduce complicated precedence behavior, and simple alternatives exist for users.

Additionally, RTE aims to have defined behavior in all circumstances and to document and prove that behavior with unit tests.

### limiting state space

Many modern routers coordinate by mutating a common data structure -- unsurprisingly, usually called a `Router`. In larger applications that pass around the routers and subrouters, setting different flags and options in different locations, the state of the router can be at best hard to reason about and at worst undefined -- it is not uncommon for certain feature combinations to fail or have unexpected results.

By centering on the simple `rte.Route` and not exposing any mutable state, RTE keeps its interface simple to understand, while also simplifying its own internals. Because most routing libraries focus on the mutable router object, they do not have explicit finalization, and thus their internal logic must remain open to modification at any time -- RTE does not have this problem.

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
