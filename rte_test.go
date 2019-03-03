package rte_test

import (
	"encoding/json"
	"fmt"
	"github.com/jwilner/rte"
	"github.com/jwilner/rte/internal/funcs"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
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
			Name:   "name unrequired",
			Routes: rte.Routes("GET /:", func(w http.ResponseWriter, r *http.Request, a string) {}),
		},
		{
			Name:    "invalidSegmentInvalidChar",
			Routes:  rte.Routes("GET /*", func(w http.ResponseWriter, r *http.Request) {}),
			WantErr: true,
			ErrType: rte.ErrTypeInvalidSegment,
			ErrIdx:  0,
			ErrMsg:  `route 0 "GET /*": invalid segment`,
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
				"GET /:whoo", func(w http.ResponseWriter, r *http.Request, _, _ string) {},
			),
			WantErr: true,
			ErrType: rte.ErrTypeParamCountMismatch,
			ErrIdx:  0,
			ErrMsg:  `route 0 "GET /:whoo": path and handler have different numbers of parameters`,
		},
		{
			Name: "method all can drop",
			Routes: rte.Routes(
				rte.Prefix("/:whoo", rte.Routes(
					"GET", func(w http.ResponseWriter, r *http.Request, whoo string) {

					},
					rte.MethodAny, func(w http.ResponseWriter, r *http.Request) {
						// the number of handler parameters is fewer than path parameters -- special case for MethodAny
					},
				)),
			),
		},
		{
			Name: "method all cannot exceed",
			Routes: rte.Routes(
				rte.Prefix("/:whoo", rte.Routes(
					"GET", func(w http.ResponseWriter, r *http.Request, whoo string) {

					},
					rte.MethodAny, func(w http.ResponseWriter, r *http.Request, whoo, whee string) {
					},
				)),
			),
			WantErr: true,
			ErrType: rte.ErrTypeParamCountMismatch,
			ErrIdx:  1,
			ErrMsg:  `route 1 "~ /:whoo": path and handler have different numbers of parameters`,
		},
		{
			Name: "excessively long path",
			Routes: rte.Routes(
				"GET "+strings.Repeat("/:whoo", len(funcs.PathVars{})+1),
				func(w http.ResponseWriter, r *http.Request, _ [8]string) {

				},
			),
			WantErr: true,
			ErrType: rte.ErrTypeOutOfRange,
			ErrIdx:  0,
			ErrMsg: `route 0 "GET ` + strings.Repeat("/:whoo", len(funcs.PathVars{})+1) +
				`": path has more than ` + strconv.Itoa(len(funcs.PathVars{})) + ` parameters`,
		},
	} {
		t.Run(c.Name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("panicked: %v", p)
				}
			}()
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
					t.Fatalf("expected error type %v, but got %v", c, e)
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
		name, skipReason string
		req              *http.Request
		rte              []rte.Route
		code             int
		body             string
	}{
		{
			name: "match",
			req:  httptest.NewRequest("GET", "/abc", nil),
			rte:  rte.Routes("GET /abc", h200),
			code: 200, body: "null",
		},
		{
			name: "wrong-method",
			req:  httptest.NewRequest("PUT", "/abcd", nil),
			rte:  rte.Routes("POST /abcd", h200),
			code: 404, body: "404",
		},
		{
			name: "match-trailing",
			req:  httptest.NewRequest("HEAD", "/abc/", nil),
			rte:  rte.Routes("HEAD /abc/", h200),
			code: 200, body: "null",
		},
		{
			name: "require-trailing",
			req:  httptest.NewRequest("GET", "/abc/", nil),
			rte:  rte.Routes("GET /abc", h200),
			code: 404, body: "404",
		},
		{
			name: "nested-miss",
			req:  httptest.NewRequest("GET", "/abc/abcde", nil),
			rte:  rte.Routes("GET /abc/abcdef", h200),
			code: 404, body: "404",
		},
		{
			name: "unequal",
			req:  httptest.NewRequest("GET", "/abc/abcdeg24", nil),
			rte:  rte.Routes("GET /abc/abcdef", h200),
			code: 404, body: "404",
		},
		{
			name: "slash-match",
			req:  httptest.NewRequest("GET", "/", nil),
			rte:  rte.Routes("GET /", h200),
			code: 200, body: "null",
		},
		{
			name: "wildcard-match",
			req:  httptest.NewRequest("GET", "/abc", nil),
			rte: rte.Routes(
				"GET /:whoo",
				func(w http.ResponseWriter, r *http.Request, whoo string) {
					_ = json.NewEncoder(w).Encode([]string{whoo})
				},
			),
			code: 200, body: `["abc"]`,
		},
		{
			name: "multiple-wildcard",
			req:  httptest.NewRequest("GET", "/abc/123", nil),
			rte: rte.Routes(
				"GET /:foo/:bar",
				func(w http.ResponseWriter, r *http.Request, foo, bar string) {
					_ = json.NewEncoder(w).Encode([]string{foo, bar})
				},
			),
			code: 200, body: `["abc","123"]`,
		},
		{
			name: "match-method-not-allowed",
			req:  httptest.NewRequest("GET", "/abc/123", nil),
			rte: rte.Routes(
				rte.MethodAny+" /:foo/:bar",
				func(w http.ResponseWriter, r *http.Request, foo, bar string) {
					w.WriteHeader(http.StatusMethodNotAllowed)
					_ = json.NewEncoder(w).Encode([]string{foo, bar})
				},
			),
			code: 405, body: `["abc","123"]`,
		},
		{
			req: httptest.NewRequest("GET", "/abc/123", nil),
			rte: rte.Routes(
				rte.MethodAny+" /:foo/:bar",
				func(w http.ResponseWriter, r *http.Request, foo, bar string) {
					w.WriteHeader(http.StatusMethodNotAllowed)
					_ = json.NewEncoder(w).Encode([]string{foo, bar})
				},
			),
			code: 405, body: `["abc","123"]`,
		},

		// multi route
		{
			req: httptest.NewRequest("GET", "/abc/123", nil),
			rte: rte.Routes(
				"GET /abc/:bar",
				func(w http.ResponseWriter, r *http.Request, bar string) {
					w.WriteHeader(http.StatusAccepted)
					_ = json.NewEncoder(w).Encode([]string{bar})
				},
				"GET /abc", h200,
			),
			code: http.StatusAccepted, body: `["123"]`,
		},
		{
			name: "wildcard margin",
			req:  httptest.NewRequest("GET", "/foo/g", nil),
			rte: rte.Routes(
				"GET /foo/bar/baz", h200,
				"GET /foo/:foo_id", func(w http.ResponseWriter, r *http.Request, fooID string) {
					_ = json.NewEncoder(w).Encode([]string{fooID})
				},
			),
			code: 200, body: `["g"]`,
		},
		{
			name:       "wildcard shadowing",
			skipReason: "knowon failure",
			req:        httptest.NewRequest("GET", "/foo/bar", nil),
			rte: rte.Routes(
				"GET /foo/bar/baz", h200,
				"GET /foo/:foo_id", func(w http.ResponseWriter, r *http.Request, fooID string) {
					_ = json.NewEncoder(w).Encode([]string{fooID})
				},
			),
			code: 200, body: `["bar"]`,
		},
		{
			name: "github example",
			req:  httptest.NewRequest("GET", "/users/blah/received_events", nil),
			rte: rte.Routes(
				"GET /authorizations", h200,
				"GET /authorizations/:id", h200,
				"POST /authorizations", h200,
				"PUT /authorizations/clients/:client_id", h200,
				"PATCH /authorizations/:id", h200,
				"DELETE /authorizations/:id", h200,
				"GET /applications/:client_id/tokens/:access_token", h200,
				"DELETE /applications/:client_id/tokens", h200,
				"DELETE /applications/:client_id/tokens/:access_token", h200,
				"GET /events", h200,
				"GET /repos/:owner/:repo/events", h200,
				"GET /networks/:owner/:repo/events", h200,
				"GET /orgs/:org/events", h200,
				"GET /users/:user/received_events", func(w http.ResponseWriter, r *http.Request, id string) {
					_, _ = fmt.Fprintln(w, id)
				},
				"GET /users/:user/received_events/public", h200,
				"GET /users/:user/events", h200,
				"GET /users/:user/events/public", h200,
				"GET /users/:user/events/orgs/:org", h200,
			),
			code: 200, body: "blah",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			tbl := rte.Must(tt.rte)
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

func TestParseVars(t *testing.T) {
	cases := []struct {
		Name       string
		Request    *http.Request
		Routes     []rte.Route
		Expected   []string
		ExpectedOK bool
	}{
		{
			"empty",
			httptest.NewRequest("GET", "/blah", nil),
			rte.Routes("GET /blah", func(http.ResponseWriter, *http.Request) {}),
			[]string{},
			true,
		},
		{
			"single",
			httptest.NewRequest("GET", "/blah", nil),
			rte.Routes("GET /:abc", func(http.ResponseWriter, *http.Request) {}),
			[]string{"blah"},
			true,
		},
		{
			"multi",
			httptest.NewRequest("GET", "/blah/abc/bar", nil),
			rte.Routes("GET /:abc/abc/:def", func(http.ResponseWriter, *http.Request) {}),
			[]string{"blah", "bar"},
			true,
		},
		{
			"after-start",
			httptest.NewRequest("GET", "/abc/bar", nil),
			rte.Routes("GET /abc/:def", func(http.ResponseWriter, *http.Request) {}),
			[]string{"bar"},
			true,
		},
		{
			"no-match",
			httptest.NewRequest("GET", "/abcd/bar", nil),
			rte.Routes("GET /abc/:def", func(http.ResponseWriter, *http.Request) {}),
			[]string{},
			false,
		},
		{
			"partial",
			httptest.NewRequest("GET", "/abc/", nil),
			rte.Routes("GET /:def/123/", func(http.ResponseWriter, *http.Request) {}),
			[]string{"abc"},
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			tbl := rte.Must(c.Routes)
			res, ok := tbl.Vars(c.Request)

			if !reflect.DeepEqual(c.Expected, res) {
				t.Fatalf("Expected %#v but got %#v", c.Expected, res)
			}
			if ok != c.ExpectedOK {
				t.Fatalf("Expected ok %v", c.ExpectedOK)
			}
		})
	}
}
