package rte_test

import (
	"fmt"
	"github.com/jwilner/rte"
	"net/http"
	"net/http/httptest"
)

func ExampleMiddlewareFunc() {
	mw := rte.MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		_, _ = fmt.Fprintln(w, "Hi!")
		next.ServeHTTP(w, r)
		_, _ = fmt.Fprintln(w, "Goodbye!")
	})

	tbl := rte.Must(rte.Routes(
		"GET /hello/:name", func(w http.ResponseWriter, r *http.Request, name string) {
			_, _ = fmt.Fprintf(w, "How are you, %v?\n", name)
		}, mw,
	))

	w := httptest.NewRecorder()
	tbl.ServeHTTP(w, httptest.NewRequest("GET", "/hello/bob", nil))

	fmt.Print(w.Body.String())

	// Output: Hi!
	// How are you, bob?
	// Goodbye!
}

func ExampleTable_ServeHTTP() {
	tbl := rte.Must(rte.Routes(
		"GET /hello/:name", func(w http.ResponseWriter, r *http.Request, name string) {
			_, _ = fmt.Fprintf(w, "Hello %v!\n", name)
		},
	))

	w := httptest.NewRecorder()
	tbl.ServeHTTP(w, httptest.NewRequest("GET", "/hello/bob", nil))

	fmt.Print(w.Body.String())

	// Output: Hello bob!
}

func ExampleRoutes() {
	routes := rte.Prefix("/my-resource", rte.Routes(
		"POST", func(w http.ResponseWriter, r *http.Request) {
			// create
		},
		rte.Prefix("/:id", rte.Routes(
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
		)),
	))

	for _, r := range routes {
		fmt.Printf("%v\n", r)
	}

	// Output: POST /my-resource
	// GET /my-resource/:id
	// PUT /my-resource/:id
	// DELETE /my-resource/:id
	// ~ /my-resource/:id
}

func ExampleOptTrailingSlash() {
	routes := rte.OptTrailingSlash(rte.Routes(
		"GET /", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, "hello world!")
		},
		"GET /:name", func(w http.ResponseWriter, r *http.Request, name string) {
			_, _ = fmt.Fprintf(w, "hello %v!\n", name)
		},
	))

	for _, r := range routes {
		fmt.Printf("%v\n", r)
	}

	// Output: GET /
	// GET /:name
	// GET /:name/
}

func ExampleMust() {
	defer func() {
		p := recover()
		fmt.Printf("panicked! %v\n", p)
	}()

	_ = rte.Must(rte.Routes(
		"GET /hello/:name", func(w http.ResponseWriter, r *http.Request) {
		},
	))

	// Output: panicked! route 0 "GET /hello/:name": path and handler have different numbers of parameters
}

func ExampleNew() {
	_, err := rte.New(rte.Routes(
		"GET /hello/:name", func(w http.ResponseWriter, r *http.Request) {
		},
	))

	fmt.Printf("errored! %v", err)

	// Output: errored! route 0 "GET /hello/:name": path and handler have different numbers of parameters
}

func ExampleError() {
	_, err := rte.New(rte.Routes(
		"GET /hello", func(w http.ResponseWriter, r *http.Request) {
		},
		"GET /hello/:name", func(w http.ResponseWriter, r *http.Request) {
		},
	))

	_, _ = fmt.Printf("%v", err.(rte.Error).Route)

	// Output: GET /hello/:name
}

func ExamplePrefix() {
	routes := rte.Prefix("/hello", rte.Routes(
		"GET /", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, "hello")
		},
	))

	for _, r := range routes {
		fmt.Printf("%v\n", r)
	}

	// Output: GET /hello/
}
