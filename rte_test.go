package rte_test

import (
	"encoding/json"
	"fmt"
	"github.com/jwilner/rte"
	"github.com/jwilner/rte/internal/funcs"
	"net/http"
	"net/http/httptest"
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
		{
			Name: "method all can drop",
			Routes: rte.Routes(
				rte.Prefix("/:whoo", rte.Routes(
					"GET", func(w http.ResponseWriter, r *http.Request, whoo string) {

					},
					rte.MethodAll, func(w http.ResponseWriter, r *http.Request) {
						// the number of handler parameters is fewer than path parameters -- special case for MethodAll
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
					rte.MethodAll, func(w http.ResponseWriter, r *http.Request, whoo, whee string) {
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

