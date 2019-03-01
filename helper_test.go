package rte_test

import (
	"fmt"
	"github.com/jwilner/rte"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type mockH bool

func (m mockH) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

type mockMW bool

func (mockMW) Handle(w http.ResponseWriter, r *http.Request, next http.Handler) {
}
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

type stringMW string

func (s stringMW) Handle(w http.ResponseWriter, r *http.Request, next http.Handler) {
	_, _ = fmt.Fprintf(w, "%s\n", s)
	next.ServeHTTP(w, r)
}

func TestGlobalMiddleware(t *testing.T) {
	mw1 := mockMW(true)
	t.Run("empty", func(t *testing.T) {
		rts := rte.GlobalMiddleware(nil, nil)
		if len(rts) != 0 {
			t.Errorf("Wanted no routes returned")
		}
	})
	t.Run("nilPassed", func(t *testing.T) {
		rts := rte.GlobalMiddleware(nil, []rte.Route{
			{Method: "GET", Path: "/"},
		})
		want := []rte.Route{{Method: "GET", Path: "/"}}
		if !reflect.DeepEqual(rts, want) {
			t.Errorf("Wanted %v but got %v", want, rts)
		}
	})
	t.Run("nilPassedMwPresent", func(t *testing.T) {
		rts := rte.GlobalMiddleware(nil, []rte.Route{
			{Method: "GET", Path: "/", Middleware: mw1},
		})
		want := []rte.Route{{Method: "GET", Path: "/", Middleware: mw1}}
		if !reflect.DeepEqual(rts, want) {
			t.Errorf("Wanted %v but got %v", want, rts)
		}
	})
	t.Run("setsMW", func(t *testing.T) {
		rts := rte.GlobalMiddleware(mw1, []rte.Route{
			{Method: "GET", Path: "/"},
		})
		want := []rte.Route{{Method: "GET", Path: "/", Middleware: mw1}}
		if !reflect.DeepEqual(rts, want) {
			t.Errorf("Wanted %v but got %v", want, rts)
		}
	})
	t.Run("composes", func(t *testing.T) {
		tbl := rte.Must(rte.GlobalMiddleware(stringMW("hi"), []rte.Route{
			{
				Method:     "GET",
				Path:       "/",
				Handler:    func(w http.ResponseWriter, r *http.Request) {},
				Middleware: stringMW("bye"),
			},
		}))

		w := httptest.NewRecorder()
		tbl.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		res := w.Body.String()
		want := "hi\nbye\n"
		if res != want {
			t.Errorf("Wanted %q but got %q", want, res)
		}
	})
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
			PanicVal: `rte.Routes: missing a target for "GET /" at argument 1`,
		},
		{
			Name: "invalid handler",
			Args: []interface{}{
				"GET /", func() {},
			},
			PanicVal: "rte.Routes: invalid handler for \"GET /\" in position 1: unknown handler type: func()",
		},
		{
			Name: "prefix shorthand",
			Args: []interface{}{
				"/resources", []rte.Route{
					{Method: "GET", Path: "/hi"},
					{Method: "POST", Path: "/bye"},
				},
			},
			WantResult: []rte.Route{
				{Method: "GET", Path: "/resources/hi"},
				{Method: "POST", Path: "/resources/bye"},
			},
		},
		{
			Name: "invalid prefix shorthand",
			Args: []interface{}{
				"POST", []rte.Route{
					{Method: "GET", Path: "/hi"},
					{Method: "POST", Path: "/bye"},
				},
			},
			PanicVal: "rte.Routes: if providing []Route as a target, reqLine must be a prefix",
		},
		{
			Name: "prefix shorthand middleware",
			Args: []interface{}{
				"/resources", []rte.Route{
					{Method: "GET", Path: "/hi"},
					{Method: "POST", Path: "/bye"},
				}, mw,
			},
			WantResult: []rte.Route{
				{Method: "GET", Path: "/resources/hi", Middleware: mw},
				{Method: "POST", Path: "/resources/bye", Middleware: mw},
			},
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
