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

		bCtx := BindContext{badArgument: defaultBadArgument}
		for i, seg := range strings.SplitAfter(r.Path, "/")[1:] {
			// normalize
			norm, name, err := normalize(seg)
			if err != nil {
				return nil, fmt.Errorf("route %v: invalid segment: %v", i, err)
			}

			// we've got a variable
			if name != "" {
				bCtx.ParamPos = append(bCtx.ParamPos, i+1) // account for first segment we skipped
				bCtx.ParamNames = append(bCtx.ParamNames, name)
			}

			if n.children[norm] == nil {
				n.children[norm] = &node{children: make(map[string]*node)}
			}
			n = n.children[norm]
		}

		if n.h != nil {
			return nil, fmt.Errorf("route %v: already has a handler for %v %#v", i, r.Method, r.Path)
		}

		n.h = r.Handler

		if r.Middleware != nil {
			n.h = r.Middleware(n.h)
		}
	}

	t.Default = http.NotFoundHandler()

	return t, nil
}

// BindContext provides context about the bound route to the handler.
type BindContext struct {
	ParamPos    []int
	ParamNames  []string
	badArgument func(w http.ResponseWriter, r *http.Request, pos int, err error)
}

// BadArgument should be called by bound handlers to report a bad argument and return a 400 to the client.
func (b *BindContext) BadArgument(w http.ResponseWriter, r *http.Request, pos int, err error) {
	b.badArgument(w, r, pos, err)
}

func defaultBadArgument(w http.ResponseWriter, _ *http.Request, _ int, _ error) {
	w.WriteHeader(http.StatusBadRequest)
}

func normalize(seg string) (norm string, name string, err error) {
	switch {
	case strings.ContainsAny(seg, "*"):
		err = fmt.Errorf("segment %q contains invalid characters", seg)
	case seg == "", seg[0] != ':':
		norm = seg
	case seg == ":", seg == ":/":
		err = fmt.Errorf("wildcard segment %q must have a name", seg)
	case seg[len(seg)-1] == '/':
		// trim off colon and slash for name
		norm, name = "*/", seg[1:len(seg)-1]
	default:
		// trim off colon for name
		norm, name = "*", seg[1:]
	}
	return
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

	var (
		i      int
		params [maxVars]string
	)
	// Analogous to `SplitAfter`, but avoids an alloc for fun
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

			var m string
			if m, n = n.match(r.URL.Path[start:end]); n == nil {
				t.Default.ServeHTTP(w, r)
				return
			} else if m != "" { // m is a path var
				params[i] = m
				i++
			}

			start = end
		}
	}

	if n.h == nil {
		t.Default.ServeHTTP(w, r)
		return
	}

	n.h(w, r, params)
}

type node struct {
	children map[string]*node
	h        BoundHandler
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
