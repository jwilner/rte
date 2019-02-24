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
	// MethodNotAllowed can be provided used as a method within a route to handle scenarios when the path but not
	// the method are matched.
	//
	// E.g., serve gets on '/foo/:foo_id' and return a 405 for everything else (405 handler can also access path vars):
	// 		_ = rte.Must([]rte.Route{
	// 			{Method: "GET", Path: "/foo/:foo_id", Handler: handlerGet},
	// 			{Method: rte.MethodNotAllowed, Path: "/foo/:foo_id", Handler: handler405},
	// 		})
	MethodNotAllowed = "405"
)

const (
	maxVars = 8
)

type pathVars [maxVars]string

// Handler is a handler function permitting no-allocation handling of path variables
type Handler func(w http.ResponseWriter, r *http.Request, pathVars pathVars)

// Middleware is shorthand for a function which takes in a handler and returns another
type Middleware = func(Handler) Handler

// Route is data for routing to a handler
type Route struct {
	Method, Path string
	Handler      Handler
	Middleware   Middleware
}

func (r Route) String() string {
	m := "<nil>"
	if r.Method != "" {
		m = r.Method
	}

	p := "<nil>"
	if r.Path != "" {
		p = r.Path
	}

	return fmt.Sprintf("%v %v", m, p)
}

const (
	// ErrTypeMethodEmpty means a route was missing a method
	ErrTypeMethodEmpty = iota
	// ErrTypeNilHandler means a route had a nil handler
	ErrTypeNilHandler
	// ErrTypePathEmpty means a path was empty
	ErrTypePathEmpty
	// ErrTypeNoInitialSlash means the path was missing the initial slash
	ErrTypeNoInitialSlash
	// ErrTypeInvalidSegment means there was an invalid segment within a path
	ErrTypeInvalidSegment
	// ErrTypeDuplicateHandler means more than one handler was provided for the same method and path.
	ErrTypeDuplicateHandler
)

// Error encapsulates table construction errors
type Error struct {
	Type, Idx int
	Route     Route
	cause     error
}

func (e Error) Error() string {
	msg := "unknown error"
	switch e.Type {
	case ErrTypeMethodEmpty:
		msg = "method cannot be empty"
	case ErrTypeNilHandler:
		msg = "handler cannot be nil"
	case ErrTypePathEmpty:
		msg = "path cannot be empty"
	case ErrTypeNoInitialSlash:
		msg = "no initial slash"
	case ErrTypeInvalidSegment:
		msg = "invalid segment"
	case ErrTypeDuplicateHandler:
		msg = "duplicate handler"
	}

	if e.cause != nil {
		return fmt.Sprintf("route %d %q: %v: %v", e.Idx, e.Route, msg, e.cause)
	}

	return fmt.Sprintf("route %d %q: %v", e.Idx, e.Route, msg)
}

// Cause returns the causing error or nil
func (e Error) Cause() error {
	return e.cause
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
			return nil, Error{Type: ErrTypeMethodEmpty, Idx: i, Route: r}
		}

		if r.Handler == nil {
			return nil, Error{Type: ErrTypeNilHandler, Idx: i, Route: r}
		}

		if r.Path == "" {
			return nil, Error{Type: ErrTypePathEmpty, Idx: i, Route: r}
		}

		if r.Path[0] != '/' {
			return nil, Error{Type: ErrTypeNoInitialSlash, Idx: i, Route: r}
		}

		n := t.root
		for _, seg := range strings.SplitAfter(r.Path, "/")[1:] {
			// normalize
			seg, err := normalize(seg)
			if err != nil {
				return nil, Error{Type: ErrTypeInvalidSegment, Idx: i, Route: r, cause: err}
			}

			if n.children[seg] == nil {
				n.children[seg] = newNode()
			}

			n = n.children[seg]
		}

		if _, has := n.methods[r.Method]; has {
			return nil, Error{Type: ErrTypeDuplicateHandler, Idx: i, Route: r}
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
		methods:  make(map[string]Handler),
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

	if h, ok := node.methods[MethodNotAllowed]; ok {
		h(w, r, params)
		return
	}

	t.Default.ServeHTTP(w, r)
}

type node struct {
	children map[string]*node
	methods  map[string]Handler
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
