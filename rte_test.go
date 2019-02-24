package rte_test

import (
	"encoding/json"
	"github.com/jwilner/rte"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_matchPath(t *testing.T) {
	var h200 = func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(nil)
	}
	var h404 = func(w http.ResponseWriter, r *http.Request) {
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
			rte.Route{Method: "GET", Path: "/abc", Handler: rte.Func(h200)},
			200, "null",
		},
		{
			"wrong-method",
			httptest.NewRequest("PUT", "/abcd", nil),
			rte.Route{Method: "POST", Path: "/abcd", Handler: rte.Func(h200)},
			404, "404",
		},
		{
			"match-trailing",
			httptest.NewRequest("HEAD", "/abc/", nil),
			rte.Route{Method: "HEAD", Path: "/abc/", Handler: rte.Func(h200)},
			200, "null",
		},
		{
			"require-trailing",
			httptest.NewRequest("GET", "/abc/", nil),
			rte.Route{Method: "GET", Path: "/abc", Handler: rte.Func(h200)},
			404, "404",
		},
		{
			"slash-match",
			httptest.NewRequest("GET", "/", nil),
			rte.Route{Method: "GET", Path: "/", Handler: rte.Func(h200)},
			200, "null",
		},
		{
			"wildcard-match",
			httptest.NewRequest("GET", "/abc", nil),
			rte.Route{
				Method: "GET", Path: "/:whoo",
				Handler: rte.Func1(func(w http.ResponseWriter, r *http.Request, whoo string) {
					_ = json.NewEncoder(w).Encode([]string{whoo})
				}),
			},
			200, `["abc"]`,
		},
		{
			"multiple-wildcard",
			httptest.NewRequest("GET", "/abc/123", nil),
			rte.Route{
				Method: "GET", Path: "/:foo/:bar",
				Handler: rte.Func2(func(w http.ResponseWriter, r *http.Request, foo, bar string) {
					_ = json.NewEncoder(w).Encode([]string{foo, bar})
				}),
			},
			200, `["abc","123"]`,
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
