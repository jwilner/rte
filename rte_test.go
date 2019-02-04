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
	h200 := func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rte.PathVars(r))
	}
	h404 := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("404"))
	}

	tests := []struct {
		name string
		req  *http.Request
		rte  rte.Route
		body string
	}{
		{
			"match",
			httptest.NewRequest("GET", "/abc", nil),
			rte.Func("GET", "/abc", h200),
			"null",
		},
		{
			"wrong-method",
			httptest.NewRequest("PUT", "/abcd", nil),
			rte.Func("POST", "/abcd", h200),
			"404",
		},
		{
			"match-trailing",
			httptest.NewRequest("HEAD", "/abc/", nil),
			rte.Func("HEAD", "/abc/", h200),
			"null",
		},
		{
			"require-trailing",
			httptest.NewRequest("GET", "/abc/", nil),
			rte.Func("GET", "/abc", h200),
			"404",
		},
		{
			"slash-match",
			httptest.NewRequest("GET", "/", nil),
			rte.Func("GET", "/", h200),
			"null",
		},
		{
			"wildcard-match",
			httptest.NewRequest("GET", "/abc", nil),
			rte.Func("GET", "/*", h200),
			`["abc"]`,
		},
		{
			"multiple-wildcard",
			httptest.NewRequest("GET", "/abc/123", nil),
			rte.Func("GET", "/*/*", h200),
			`["abc","123"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl := rte.Must(tt.rte)
			tbl.Default = http.HandlerFunc(h404)

			w := httptest.NewRecorder()
			tbl.ServeHTTP(w, tt.req)

			if body := strings.TrimSpace(w.Body.String()); body != tt.body {
				t.Errorf("resp: got %#v, want %#v", body, tt.body)
			}
		})
	}
}
