// Package rte provides simple, performant routing.
// - Define individual routes with `rte.Func` and generated siblings
// - Combine them into a table with `rte.Must` or `rte.New`
package rte

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	errTooManyParams = "path has too many parameters"
)

var (
	// ErrWrongNumParams is returned when a Binder attempts to wrap a hand
	ErrWrongNumParams = errors.New(errTooManyParams)
)

// Binder is the common interface which binding handlers have to fulfill
type Binder interface {
	// Bind is invoked with segment indexes corresponding to path wildcards and returns a handler or an error.
	Bind(segIdxes []int) (http.HandlerFunc, error)
}

// Func routes requests matching the method and path to a handler
func Func(method, path string, f http.HandlerFunc) Route {
	return Bind(method, path, func0(f))
}

type func0 http.HandlerFunc

func (f func0) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 0 {
		return nil, ErrWrongNumParams
	}
	return http.HandlerFunc(f), nil
}

// Bind takes in a method, path, and a Binder and returns a route.
func Bind(method, path string, f Binder) Route {
	return Route{Method: method, Path: path, Handler: f}
}

// Route is data for routing to a handler
type Route struct {
	Method, Path string
	Handler      Binder
}

// Must builds routes into a Table and panics if there's an error
func Must(routes ...Route) *Table {
	t, e := New(routes...)
	if e != nil {
		panic(e.Error())
	}
	return t
}

// New builds routes into a Table or returns an error
func New(routes ...Route) (*Table, error) {
	t := new(Table)

	t.m = make(map[string]*node)

	for i, r := range routes {
		if r.Method == "" {
			return nil, fmt.Errorf("route %v: Method cannot be empty", i)
		}

		if r.Handler == nil {
			return nil, fmt.Errorf("route %v: handle cannot be nil", i)
		}

		if r.Path == "" {
			return nil, fmt.Errorf("route %v: Path cannot be empty", i)
		}

		if r.Path[0] != '/' {
			return nil, fmt.Errorf("route %v: must start with / -- got %q", i, r.Path)
		}

		if t.m[r.Method] == nil {
			t.m[r.Method] = &node{children: make(map[string]*node)}
		}

		n := t.m[r.Method]

		var paramPos []int
		for i, seg := range strings.SplitAfter(r.Path, "/")[1:] {
			// normalize
			var err error
			if seg, err = normalize(seg); err != nil {
				return nil, fmt.Errorf("route %v: invalid segment: %v", i, err)
			} else if seg == "*" || seg == "*/" {
				paramPos = append(paramPos, i+1) // account for first segment we skipped
			}

			if n.children[seg] == nil {
				n.children[seg] = &node{children: make(map[string]*node)}
			}
			n = n.children[seg]
		}

		if n.h != nil {
			return nil, fmt.Errorf("route %v: already has a handler for %v %#v", i, r.Method, r.Path)
		}

		var err error
		if n.h, err = r.Handler.Bind(paramPos); err != nil {
			return nil, fmt.Errorf("route %v: invalid parameters: %v", i, err)
		}
	}

	t.Default = http.NotFoundHandler()

	return t, nil
}

func normalize(seg string) (string, error) {
	switch {
	case strings.ContainsAny(seg, "*"):
		return "", fmt.Errorf("segment %q contains invalid characters", seg)
	case seg == "", seg[0] != ':':
		return seg, nil
	case seg == ":", seg == ":/":
		return "", fmt.Errorf("wildcard segment %q must have a name", seg)
	case seg[len(seg)-1] == '/':
		return "*/", nil
	default:
		return "*", nil
	}
}

// Table manages the routing table and a default handler
type Table struct {
	m       map[string]*node
	Default http.Handler
}

func (t *Table) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if t.m[r.Method] == nil {
		t.Default.ServeHTTP(w, r)
		return
	}

	n := t.m[r.Method]

	// Analogus to `SplitAfter`, but avoids an alloc for fun
	// "" -> [], "/" -> [""], "/abc" -> ["/", "abc"], "/abc/" -> [",", "abc/", ""]
	if start := strings.Index(r.URL.Path, "/") + 1; start != 0 {
		for hitEnd := false; !hitEnd; {
			var end int
			if offset := strings.Index(r.URL.Path[start:], "/"); offset != -1 {
				end = start + offset + 1
			} else {
				end = len(r.URL.Path)
				hitEnd = true
			}

			if _, n = n.match(r.URL.Path[start:end]); n == nil {
				t.Default.ServeHTTP(w, r)
				return
			}

			start = end
		}
	}

	if n.h == nil {
		t.Default.ServeHTTP(w, r)
		return
	}

	n.h.ServeHTTP(w, r)
}

type node struct {
	children map[string]*node
	h        http.Handler
}

func (n *node) match(seg string) (string, *node) {
	if c := n.children[seg]; c != nil {
		return "", c
	} else if l := len(seg) - 1; l >= 0 && seg[l] == '/' {
		return seg[:l], n.children["*/"]
	} else {
		return seg, n.children["*"]
	}
}

// ("/abc/def", [0]) -> [""]
// ("/abc/def", [1]) -> ["abc"]
// ("/abc/def", [2]) -> ["def"]
// ("/abc/def", [0, 2]) -> ["", "def"]
// ("/abc/", [1]) -> panic
func findNSegments(path string, segIdxes []int, segs []string) {
	var (
		curSegIdx    = 0
		posLastSlash = 0
		offsetSlash  = strings.IndexByte(path, '/')
	)

	for slashNum := 0; curSegIdx < len(segIdxes); slashNum++ {
		if segIdxes[curSegIdx] == slashNum {
			if offsetSlash == -1 {
				segs[curSegIdx] = path[posLastSlash:]
			} else {
				segs[curSegIdx] = path[posLastSlash : posLastSlash+offsetSlash] // don't include slash
			}
			curSegIdx++
		} else if offsetSlash == -1 {
			panic("Ran off the end")
		}

		posNextSlash := posLastSlash + offsetSlash + 1
		posLastSlash, offsetSlash = posNextSlash, strings.IndexByte(path[posNextSlash:], '/')
	}
}
