package rte_test

import (
	"github.com/jwilner/rte"
	"reflect"
	"testing"
)

func TestOptTrailingSlash(t *testing.T) {
	for _, tt := range []struct {
		name     string
		in, want []rte.Route
	}{
		{
			"empty",
			nil,
			nil,
		},
		{
			"addsNoSlash",
			[]rte.Route{{Method: "GET", Path: "/hi"}},
			[]rte.Route{{Method: "GET", Path: "/hi"}, {Method: "GET", Path: "/hi/"}},
		},
		{
			"addsSlash",
			[]rte.Route{{Method: "GET", Path: "/hi/"}},
			[]rte.Route{{Method: "GET", Path: "/hi/"}, {Method: "GET", Path: "/hi"}},
		},
		{
			"unchanged",
			[]rte.Route{{Method: "GET", Path: "/hi/"}, {Method: "GET", Path: "/hi"}},
			[]rte.Route{{Method: "GET", Path: "/hi/"}, {Method: "GET", Path: "/hi"}},
		},
		{
			"addsJustOneIfDupe",
			[]rte.Route{{Method: "GET", Path: "/hi/"}, {Method: "GET", Path: "/hi/"}},
			[]rte.Route{{Method: "GET", Path: "/hi/"}, {Method: "GET", Path: "/hi"}, {Method: "GET", Path: "/hi/"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := rte.OptTrailingSlash(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OptTrailingSlash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrefix(t *testing.T) {
	for _, tt := range []struct {
		name, prefix string
		in, want     []rte.Route
	}{
		{
			"empty",
			"/my-prefix",
			nil,
			nil,
		},
		{
			"adds",
			"/my-prefix",
			[]rte.Route{{Path: "/hi"}},
			[]rte.Route{{Path: "/my-prefix/hi"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := rte.Prefix(tt.prefix, tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Prefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultMethod(t *testing.T) {
	m, m1 := mockH(true), mockH(false)
	for _, tt := range []struct {
		name     string
		handler  interface{}
		in, want []rte.Route
	}{
		{
			name:    "empty",
			handler: m,
		},
		{
			name:    "simple",
			handler: m,
			in:      rte.Routes("GET /", m1),
			want:    rte.Routes("GET /", m1, "~ /", m),
		},
		{
			name:    "multi-path",
			handler: m,
			in:      rte.Routes("GET /", m1, "POST /foobar", m1),
			want:    rte.Routes("GET /", m1, "~ /", m, "POST /foobar", m1, "~ /foobar", m),
		},
		{
			name:    "multi-method",
			handler: m,
			in:      rte.Routes("GET /", m1, "POST /", m1),
			want:    rte.Routes("GET /", m1, "~ /", m, "POST /", m1),
		},
		{
			name:    "no-clobber",
			handler: m,
			in:      rte.Routes("GET /", m1, "~ /", m1),
			want:    rte.Routes("GET /", m1, "~ /", m1),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := rte.DefaultMethod(tt.handler, tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Prefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
