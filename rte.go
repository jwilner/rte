// Package rte provides simple, performant routing.
// - Define individual routes with `rte.Func` and generated siblings
// - Combine them into a table with `rte.Must` or `rte.New`
package rte

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	RouteMethodNotAllowed = "405"
)

const (
	maxVars = 8
)

type pathVars [maxVars]string

// BoundHandler is a handler function permitting no-allocation handling of path variables
type BoundHandler func(w http.ResponseWriter, r *http.Request, pathVars pathVars)

// Middleware is shorthand for a function which takes in a handler and returns another
type Middleware = func(BoundHandler) BoundHandler

// Route is data for routing to a handler
type Route struct {
	Method, Path string
	Handler      BoundHandler
	Middleware   Middleware
}

// Must builds routes into a Table and panics if there's an error
func Must(routes []Route) *Table {
	t, e := New(routes)
	if e != nil {
		panic(e.Error())
	}
	return t
}

// New builds routes into a Table or returns an error
func New(routes []Route) (*Table, error) {
	t := new(Table)
	t.root = newNode()

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

		n := t.root
		for i, seg := range strings.SplitAfter(r.Path, "/")[1:] {
			// normalize
			seg, err := normalize(seg)
			if err != nil {
				return nil, fmt.Errorf("route %v: invalid segment: %v", i, err)
			}

			if n.children[seg] == nil {
				n.children[seg] = newNode()
			}

			n = n.children[seg]
		}

		if _, has := n.methods[r.Method]; has {
			return nil, fmt.Errorf("route %v: already has a handler for %v %#v", i, r.Method, r.Path)
		}

		h := r.Handler
		if r.Middleware != nil {
			h = r.Middleware(h)
		}

		n.methods[r.Method] = h
	}

	t.Default = http.NotFoundHandler()

	return t, nil
}

func newNode() *node {
	return &node{
		children: make(map[string]*node),
		methods:  make(map[string]BoundHandler),
	}
}

func normalize(seg string) (norm string, err error) {
	switch {
	case strings.ContainsAny(seg, "*"):
		err = fmt.Errorf("segment %q contains invalid characters", seg)
	case seg == "", seg[0] != ':':
		norm = seg
	case seg == ":", seg == ":/":
		err = fmt.Errorf("wildcard segment %q must have a name", seg)
	case seg[len(seg)-1] == '/':
		norm = "*/"
	default:
		norm = "*"
	}
	return
}

// Table manages the routing table and a default handler
type Table struct {
	Default http.Handler
	root    *node
}

func (t *Table) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		i      int
		params [maxVars]string
		node   = t.root
	)
	// Analogous to `SplitAfter`, but avoids an alloc for fun
	// "" -> [], "/" -> [""], "/abc" -> ["/", "abc"], "/abc/" -> ["/", "abc/", ""]
	if start := strings.IndexByte(r.URL.Path, '/') + 1; start != 0 {
		for hitEnd := false; !hitEnd; {
			var end int
			if offset := strings.IndexByte(r.URL.Path[start:], '/'); offset != -1 {
				end = start + offset + 1
			} else {
				end = len(r.URL.Path)
				hitEnd = true
			}

			var pVarName string
			if pVarName, node = node.match(r.URL.Path[start:end]); node == nil {
				t.Default.ServeHTTP(w, r)
				return
			} else if pVarName != "" { // we've matched a path var
				params[i] = pVarName
				i++
			}

			start = end
		}
	}

	if h, ok := node.methods[r.Method]; ok {
		h(w, r, params)
		return
	}

	if h, ok := node.methods[RouteMethodNotAllowed]; ok {
		h(w, r, params)
		return
	}

	t.Default.ServeHTTP(w, r)
}

type node struct {
	children map[string]*node
	methods  map[string]BoundHandler
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
