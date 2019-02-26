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
