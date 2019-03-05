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
	routes := rte.Routes(
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
				rte.MethodAny, func(w http.ResponseWriter, r *http.Request, id string) {
					// serve a 405
				},
			),
		),
	)

	for _, r := range routes {
		fmt.Printf("%v\n", r)
	}

	// Output: POST /my-resource
	// GET /my-resource/:id
	// PUT /my-resource/:id
	// DELETE /my-resource/:id
	// ~ /my-resource/:id
}

func ExampleRoutes_second() {
	mw := stringMW("abc")
	rts := rte.Routes(
		nil,

		"GET", func(w http.ResponseWriter, r *http.Request) {},
		"GET /", func(w http.ResponseWriter, r *http.Request) {},
		"/", func(w http.ResponseWriter, r *http.Request) {},
		"", func(w http.ResponseWriter, r *http.Request) {},

		"GET", func(w http.ResponseWriter, r *http.Request) {}, mw,
		"GET /", func(w http.ResponseWriter, r *http.Request) {}, mw,
		"/", func(w http.ResponseWriter, r *http.Request) {}, mw,
		"", func(w http.ResponseWriter, r *http.Request) {}, mw,

		rte.Route{Method: "OPTIONS", Path: "/bob"},
		[]rte.Route{
			{Method: "OPTIONS", Path: "/bob"},
			{Method: "BLAH", Path: "/jane"},
		},

		"/pre", []rte.Route{
			{Method: "GET", Path: "/bob"},
			{Method: "POST", Path: "/bob/hi"},
		},
		"/pre2", []rte.Route{
			{Method: "GET", Path: "/bob"},
			{Method: "POST", Path: "/bob/hi"},
		}, mw,
	)

	for _, r := range rts {
		fmt.Println(r)
	}

	// Output: GET <nil>
	// GET /
	// <nil> /
	// <nil> <nil>
	// GET <nil>
	// GET /
	// <nil> /
	// <nil> <nil>
	// OPTIONS /bob
	// OPTIONS /bob
	// BLAH /jane
	// GET /pre/bob
	// POST /pre/bob/hi
	// GET /pre2/bob
	// POST /pre2/bob/hi
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
		"GET /hello/:name", func(w http.ResponseWriter, r *http.Request, a, b string) {
		},
	))

	// Output: panicked! route 0 "GET /hello/:name": path and handler have different numbers of parameters
}

func ExampleNew() {
	_, err := rte.New(rte.Routes(
		"GET /hello/:name", func(w http.ResponseWriter, r *http.Request, a, b string) {
		},
	))

	fmt.Printf("errored! %v", err)

	// Output: errored! route 0 "GET /hello/:name": path and handler have different numbers of parameters
}

func ExampleTableError() {
	_, err := rte.New(rte.Routes(
		"GET /hello", func(w http.ResponseWriter, r *http.Request) {
		},
		"GET /hello/:name", func(w http.ResponseWriter, r *http.Request, a, b string) {
		},
	))

	_, _ = fmt.Printf("%v", err.(*rte.TableError).Route)

	// Output: GET /hello/:name
}

func ExampleTable_Vars() {
	var tbl *rte.Table
	tbl = rte.Must(rte.Routes(
		"GET /:a/:b", func(w http.ResponseWriter, r *http.Request) { // zero params can match any path
			vars, _ := tbl.Vars(r)
			for _, v := range vars {
				_, _ = fmt.Println(v)
			}
		},
	))

	tbl.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/abc/def", nil))

	// Output: abc
	// def
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

func ExampleDefaultMethod() {
	hndlr := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = fmt.Fprintf(w, "%v %v not allowed", r.Method, r.URL.Path)
	}
	routes := rte.DefaultMethod(hndlr, rte.Routes(
		"GET /foo", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, "GET /foo succeeded")
		},
		"POST /bar", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, "POST /bar succeeded")
		},
	))

	for _, r := range routes {
		fmt.Printf("%v\n", r)
	}

	tbl := rte.Must(routes)
	{
		w := httptest.NewRecorder()
		tbl.ServeHTTP(w, httptest.NewRequest("GET", "/foo", nil))
		fmt.Println(w.Body.String())
	}
	{
		w := httptest.NewRecorder()
		tbl.ServeHTTP(w, httptest.NewRequest("PRETEND", "/foo", nil))
		fmt.Println(w.Body.String())
	}

	// Output: GET /foo
	// ~ /foo
	// POST /bar
	// ~ /bar
	// GET /foo succeeded
	// PRETEND /foo not allowed
}

func ExampleWrap() {
	// applied to the one
	m1 := stringMW("and this is m1")
	// applied to both
	m2 := stringMW("this is m2")

	tbl := rte.Must(rte.Wrap(m2, rte.Routes(
		"GET /", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, "handling GET /\n")
		}, m1,
		"POST /", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, "handling POST /\n")
		},
	)))

	{
		w := httptest.NewRecorder()
		tbl.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		fmt.Print(w.Body.String())
	}

	{
		w := httptest.NewRecorder()
		tbl.ServeHTTP(w, httptest.NewRequest("POST", "/", nil))
		fmt.Print(w.Body.String())
	}

	// Output: this is m2
	// and this is m1
	// handling GET /
	// this is m2
	// handling POST /
}

func ExampleRoutes_third() {
	rts := rte.Routes(
		"GET /boo", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, "boo")
		},
	)

	var withSlashRedirects []rte.Route
	for _, route := range rts {
		target := route.Path

		c := route
		c.Path += "/"
		c.Handler = func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		}

		withSlashRedirects = append(withSlashRedirects, route, c)
	}

	tbl := rte.Must(withSlashRedirects)

	w := httptest.NewRecorder()
	tbl.ServeHTTP(w, httptest.NewRequest("GET", "/boo/", nil))
	fmt.Printf("%v %v", w.Code, w.Header().Get("Location"))

	// Output: 301 /boo
}
