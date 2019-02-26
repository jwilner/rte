package rte_test

import (
	"encoding/json"
	"fmt"
	"github.com/jwilner/rte"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	for _, c := range []struct {
		Name     string
		Routes   []rte.Route
		WantErr  bool
		ErrType  int
		ErrIdx   int
		ErrMsg   string
		CauseMsg string
	}{
		{
			Name: "emptyNoErr",
		},
		{
			Name:   "routesNoErr",
			Routes: rte.Routes("GET /", func(w http.ResponseWriter, r *http.Request) {}),
		},
		{
			Name: "methodEmpty",
			Routes: []rte.Route{
				{Method: "", Path: "/", Handler: func(w http.ResponseWriter, r *http.Request) {}},
			},
			WantErr: true,
			ErrType: rte.ErrTypeMethodEmpty,
			ErrIdx:  0,
			ErrMsg:  `route 0 "<nil> /": method cannot be empty`,
		},
		{
			Name:    "routesNilHandler",
			Routes:  []rte.Route{{Method: "GET", Path: "/", Handler: nil}},
			WantErr: true,
			ErrType: rte.ErrTypeNilHandler,
			ErrIdx:  0,
			ErrMsg:  `route 0 "GET /": handler cannot be nil`,
		},
		{
			Name:    "pathEmpty",
			Routes:  []rte.Route{{Method: "GET", Path: "", Handler: func(w http.ResponseWriter, r *http.Request) {}}},
			WantErr: true,
			ErrType: rte.ErrTypePathEmpty,
			ErrIdx:  0,
			ErrMsg:  `route 0 "GET <nil>": path cannot be empty`,
		},
		{
			Name:    "noInitialSlash",
			Routes:  rte.Routes("GET hi", func(w http.ResponseWriter, r *http.Request) {}),
			WantErr: true,
			ErrType: rte.ErrTypeNoInitialSlash,
			ErrIdx:  0,
			ErrMsg:  `route 0 "GET hi": no initial slash`,
		},
		{
			Name:    "invalidSegmentMissingName",
			Routes:  rte.Routes("GET /:", func(w http.ResponseWriter, r *http.Request) {}),
			WantErr: true,
			ErrType: rte.ErrTypeInvalidSegment,
			ErrIdx:  0,
			ErrMsg:  `route 0 "GET /:": invalid segment: wildcard segment ":" must have a name`,
		},
		{
			Name:    "invalidSegmentInvalidChar",
			Routes:  rte.Routes("GET /*", func(w http.ResponseWriter, r *http.Request) {}),
			WantErr: true,
			ErrType: rte.ErrTypeInvalidSegment,
			ErrIdx:  0,
			ErrMsg:  `route 0 "GET /*": invalid segment: segment "*" contains invalid characters`,
		},
		{
			Name: "duplicate handler",
			Routes: rte.Routes(
				"GET /", func(w http.ResponseWriter, r *http.Request) {},
				"GET /", func(w http.ResponseWriter, r *http.Request) {},
			),
			WantErr: true,
			ErrType: rte.ErrTypeDuplicateHandler,
			ErrIdx:  1,
			ErrMsg:  `route 1 "GET /": duplicate handler`,
		},
		{
			Name: "unsupported signature",
			Routes: []rte.Route{
				{Method: "GET", Path: "/:whoo", Handler: func(w http.ResponseWriter, r *http.Request, i int) {}},
			},
			WantErr: true,
			ErrType: rte.ErrTypeConversionFailure,
			ErrIdx:  0,
			ErrMsg: `route 0 "GET /:whoo": handler has an unsupported signature: unknown handler type: ` +
				`func(http.ResponseWriter, *http.Request, int)`,
			CauseMsg: `unknown handler type: func(http.ResponseWriter, *http.Request, int)`,
		},
		{
			Name: "mismatched param counts",
			Routes: rte.Routes(
				"GET /:whoo", func(w http.ResponseWriter, r *http.Request) {},
			),
			WantErr: true,
			ErrType: rte.ErrTypeParamCountMismatch,
			ErrIdx:  0,
			ErrMsg:  `route 0 "GET /:whoo": path and handler have different numbers of parameters`,
		},
	} {
		t.Run(c.Name, func(t *testing.T) {
			_, err := rte.New(c.Routes)
			if c.WantErr != (err != nil) {
				t.Fatalf("want err %v, got %v", c.WantErr, err)
			}
			if c.WantErr {
				e, ok := err.(rte.Error)
				switch {
				case !ok:
					t.Fatalf("expected a rte.Error, got %T: %v", err, err)
				case e.Type != c.ErrType:
					t.Fatalf("expected error type %v, but got %v", c.ErrType, e.Type)
				case e.Idx != c.ErrIdx:
					t.Fatalf("expected error to occur with route %v, but got route %v", c.ErrIdx, e.Idx)
				case e.Error() != c.ErrMsg:
					t.Fatalf("expected error message %v, but got %v", c.ErrMsg, e.Error())
				}

				if c.CauseMsg != "" {
					causeMsg := ""
					if e.Cause() != nil {
						causeMsg = e.Cause().Error()
					}

					if c.CauseMsg != causeMsg {
						t.Fatalf("wanted %q as a cause but got %q", c.CauseMsg, causeMsg)
					}
				}
			}
		})
	}
}

func TestMust(t *testing.T) {
	for _, c := range []struct {
		Name      string
		Routes    []rte.Route
		WantPanic bool
	}{
		{
			Name:      "emptyFine",
			Routes:    nil,
			WantPanic: false,
		},
		{
			Name:      "validRoute",
			Routes:    rte.Routes("GET /", func(w http.ResponseWriter, r *http.Request) {}),
			WantPanic: false,
		},
		{
			Name:      "invalidRoute",
			Routes:    []rte.Route{{Method: "GET", Path: "/", Handler: nil}},
			WantPanic: true,
		},
	} {
		t.Run(c.Name, func(t *testing.T) {
			defer func() {
				if p := recover(); (p != nil) != c.WantPanic {
					t.Fatalf("want panic %v but got %v", c.WantPanic, p)
				}
			}()
			rte.Must(c.Routes)
		})
	}
}

func Test_matchPath(t *testing.T) {
	h200 := func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(nil)
	}
	h404 := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte("404"))
	}

	tests := []struct {
		name string
		req  *http.Request
		rte  rte.Route
		code int
		body string
	}{
		{
			"match",
			httptest.NewRequest("GET", "/abc", nil),
			rte.Route{Method: "GET", Path: "/abc", Handler: h200},
			200, "null",
		},
		{
			"wrong-method",
			httptest.NewRequest("PUT", "/abcd", nil),
			rte.Route{Method: "POST", Path: "/abcd", Handler: h200},
			404, "404",
		},
		{
			"match-trailing",
			httptest.NewRequest("HEAD", "/abc/", nil),
			rte.Route{Method: "HEAD", Path: "/abc/", Handler: h200},
			200, "null",
		},
		{
			"require-trailing",
			httptest.NewRequest("GET", "/abc/", nil),
			rte.Route{Method: "GET", Path: "/abc", Handler: h200},
			404, "404",
		},
		{
			"slash-match",
			httptest.NewRequest("GET", "/", nil),
			rte.Route{Method: "GET", Path: "/", Handler: h200},
			200, "null",
		},
		{
			"wildcard-match",
			httptest.NewRequest("GET", "/abc", nil),
			rte.Route{
				Method: "GET", Path: "/:whoo",
				Handler: func(w http.ResponseWriter, r *http.Request, whoo string) {
					_ = json.NewEncoder(w).Encode([]string{whoo})
				},
			},
			200, `["abc"]`,
		},
		{
			"multiple-wildcard",
			httptest.NewRequest("GET", "/abc/123", nil),
			rte.Route{
				Method: "GET", Path: "/:foo/:bar",
				Handler: func(w http.ResponseWriter, r *http.Request, foo, bar string) {
					_ = json.NewEncoder(w).Encode([]string{foo, bar})
				},
			},
			200, `["abc","123"]`,
		},
		{
			"match-method-not-allowed",
			httptest.NewRequest("GET", "/abc/123", nil),
			rte.Route{
				Method: rte.MethodAll, Path: "/:foo/:bar",
				Handler: func(w http.ResponseWriter, r *http.Request, foo, bar string) {
					w.WriteHeader(http.StatusMethodNotAllowed)
					_ = json.NewEncoder(w).Encode([]string{foo, bar})
				},
			},
			405, `["abc","123"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := rte.Must([]rte.Route{tt.rte})
			tbl.Default = http.HandlerFunc(h404)

			w := httptest.NewRecorder()
			tbl.ServeHTTP(w, tt.req)

			if w.Code != tt.code {
				t.Fatalf("resp code: got %#v, want %#v", w.Code, tt.code)
			}

			if body := strings.TrimSpace(w.Body.String()); body != tt.body {
				t.Fatalf("resp: got %#v, want %#v", body, tt.body)
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	for _, c := range []struct {
		Name     string
		MW       rte.MiddlewareFunc
		WantCode int
		WantBody string
	}{
		{
			"pass-through",
			func(w http.ResponseWriter, r *http.Request, next http.Handler) {
				next.ServeHTTP(w, r)
			},
			200,
			"hullo\n",
		},
		{
			"before",
			func(w http.ResponseWriter, r *http.Request, next http.Handler) {
				_, _ = fmt.Fprintln(w, "oh hey")
				next.ServeHTTP(w, r)
			},
			200,
			"oh hey\nhullo\n",
		},
		{
			"skip",
			func(w http.ResponseWriter, r *http.Request, next http.Handler) {
				_, _ = fmt.Fprintln(w, "oh hey")
			},
			200,
			"oh hey\n",
		},
		{
			"both sides",
			func(w http.ResponseWriter, r *http.Request, next http.Handler) {
				_, _ = fmt.Fprintln(w, "oh hey")
				next.ServeHTTP(w, r)
				_, _ = fmt.Fprintln(w, "bye")
			},
			200,
			"oh hey\nhullo\nbye\n",
		},
	} {
		t.Run(c.Name, func(t *testing.T) {
			tbl := rte.Must(rte.Routes(
				"GET /",
				func(w http.ResponseWriter, r *http.Request) {
					_, _ = fmt.Fprintln(w, "hullo")
				},
				c.MW,
			))

			w := httptest.NewRecorder()
			tbl.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

			if w.Code != c.WantCode {
				t.Fatalf("Got %v, want %v", w.Code, c.WantCode)
			}

			if w.Body.String() != c.WantBody {
				t.Fatalf("Got %v, want %v", w.Body, c.WantBody)
			}
		})
	}
}

type mockH bool

func (m mockH) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

type mockMW bool

func (mockMW) Handle(w http.ResponseWriter, r *http.Request, next http.Handler) {
}

func TestRoutes(t *testing.T) {

	panics := func(t *testing.T, f func(), want interface{}) {
		defer func() {
			p := recover()
			if p == nil {
				t.Fatal("Wanted panic but didn't")
			}
			if p != want {
				t.Fatalf("Wanted panic of %q but got %q", want, p)
			}
		}()
		f()
	}

	noPanic := func(t *testing.T, f func()) {
		defer func() {
			p := recover()
			if p != nil {
				t.Fatalf("wanted no panic but got %v", p)
			}
		}()
		f()
	}

	h := mockH(true)
	mw := mockMW(true)

	for _, c := range []struct {
		Name       string
		Args       []interface{}
		PanicVal   interface{}
		WantResult []rte.Route
	}{
		{
			Name: "Empty",
		},
		{
			Name: "Inlines routes",
			Args: []interface{}{
				[]rte.Route{{Method: "POST", Path: "/"}},
				"GET /blah", h,
			},
			WantResult: []rte.Route{
				{Method: "POST", Path: "/"},
				{Method: "GET", Path: "/blah", Handler: h},
			},
		},
		{
			Name: "Inlines solitary route",
			Args: []interface{}{
				rte.Route{Method: "POST", Path: "/"},
				"GET /blah", h,
			},
			WantResult: []rte.Route{
				{Method: "POST", Path: "/"},
				{Method: "GET", Path: "/blah", Handler: h},
			},
		},
		{
			Name: "adds mw",
			Args: []interface{}{
				"GET /blah", h, mw,
				"POST /hoo", h,
			},
			WantResult: []rte.Route{
				{Method: "GET", Path: "/blah", Handler: h, Middleware: mw},
				{Method: "POST", Path: "/hoo", Handler: h},
			},
		},
		{
			Name: "skips nil",
			Args: []interface{}{
				"GET /blah", h,
				nil,
				"POST /hoo", h, mw,
			},
			WantResult: []rte.Route{
				{Method: "GET", Path: "/blah", Handler: h},
				{Method: "POST", Path: "/hoo", Handler: h, Middleware: mw},
			},
		},
		{
			Name: "method only",
			Args: []interface{}{
				"BLAH", h,
			},
			WantResult: []rte.Route{
				{Method: "BLAH", Handler: h},
			},
		},
		{
			Name: "path only",
			Args: []interface{}{
				"/blah", h,
			},
			WantResult: []rte.Route{
				{Path: "/blah", Handler: h},
			},
		},
		{
			Name: "empty method",
			Args: []interface{}{
				"", h,
			},
			WantResult: []rte.Route{
				{Handler: h},
			},
		},
		{
			Name: "not a []route or string",
			Args: []interface{}{
				23,
			},
			PanicVal: `rte.Routes: argument 0 must be either a string, a Route, or a []Route but got int: 23`,
		},
		{
			Name: "cuts off early",
			Args: []interface{}{
				"GET /",
			},
			PanicVal: `rte.Routes: missing a handler for "GET /" at argument 1`,
		},
		{
			Name: "invalid handler",
			Args: []interface{}{
				"GET /", func() {},
			},
			PanicVal: "rte.Routes: invalid handler for \"GET /\" in position 1: unknown handler type: func()",
		},
	} {
		t.Run(c.Name, func(t *testing.T) {
			if c.PanicVal != nil {
				panics(t, func() {
					rte.Routes(c.Args...)
				}, c.PanicVal)
				return
			}

			var result []rte.Route
			noPanic(t, func() {
				result = rte.Routes(c.Args...)
			})

			if !reflect.DeepEqual(result, c.WantResult) {
				t.Fatalf("results unequal: want %#v, got %#v", c.WantResult, result)
			}
		})
	}
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

	fmt.Printf("%q", routes)

	// Output: ["POST /my-resource" "GET /my-resource/:id" "PUT /my-resource/:id" "DELETE /my-resource/:id" "~ /my-resource/:id"]
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

	fmt.Printf("%q", routes)

	// Output: ["GET /" "GET /:name" "GET /:name/"]
}

func ExamplePrefix() {
	routes := rte.Prefix("/hello", rte.Routes(
		"GET /", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, "hello")
		},
	))

	fmt.Printf("%q", routes)

	// Output: ["GET /hello/"]
}
