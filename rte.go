// Package rte provides simple, performant routing.
// - Define individual routes with `rte.Func` and generated siblings
// - Combine them into a table with `rte.Must` or `rte.New`
package rte

import (
	"fmt"
	"github.com/jwilner/rte/internal/funcs"
	"net/http"
	"regexp"
	"strings"
)

const (
	// MethodAny can be provided used as a method within a route to handle scenarios when the path but not
	// the method are matched.
	//
	// E.g., serve gets on '/foo/:foo_id' and return a 405 for everything else (405 handler can also access path vars):
	// 		_ = rte.Must(rte.Routes(
	// 			"GET /foo/:foo_id", handlerGet,
	// 			rte.MethodAny + " /foo/:foo_id", handler405,
	// 		))
	MethodAny = "~"
)

// Middleware is shorthand for a function which can handle or modify a request, optionally invoke the next
// handler (or not), and modify (or set) a response.
type Middleware interface {
	Handle(w http.ResponseWriter, r *http.Request, next http.Handler)
}

// MiddlewareFunc is an adapter type permitting regular functions to be used as Middleware
type MiddlewareFunc func(w http.ResponseWriter, r *http.Request, next http.Handler)

// Handle applies wrapping behavior to a request handler
func (f MiddlewareFunc) Handle(w http.ResponseWriter, r *http.Request, next http.Handler) {
	f(w, r, next)
}

// Route is data for routing to a handler
type Route struct {
	Method, Path string
	Handler      interface{}
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
	// ErrTypeOutOfRange indicates that there are more variables in the path than this version of RTE can handle
	ErrTypeOutOfRange
	// ErrTypeDuplicateHandler means more than one handler was provided for the same method and path.
	ErrTypeDuplicateHandler
	// ErrTypeConversionFailure means that the provided value can't be converted to a handler
	ErrTypeConversionFailure
	// ErrTypeParamCountMismatch means the handler doesn't match the number of variables in the path
	ErrTypeParamCountMismatch
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
	case ErrTypeOutOfRange:
		msg = fmt.Sprintf("path has more than %v parameters", len(funcs.PathVars{}))
	case ErrTypeDuplicateHandler:
		msg = "duplicate handler"
	case ErrTypeConversionFailure:
		msg = "handler has an unsupported signature"
	case ErrTypeParamCountMismatch:
		msg = "path and handler have different numbers of parameters"
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
	t := &Table{root: newNode(""), Default: http.NotFoundHandler()}
	normalizer := regexp.MustCompile(`:[^/]*`)

	maxVars := len(funcs.PathVars{})
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

		if strings.Contains(r.Path, "*") {
			return nil, Error{Type: ErrTypeInvalidSegment, Idx: i, Route: r}
		}

		normalized := normalizer.ReplaceAllString(r.Path, "*")

		var numPathParams int
		for _, c := range normalized {
			if c == '*' {
				numPathParams++
			}
		}

		if numPathParams > maxVars {
			return nil, Error{Type: ErrTypeOutOfRange, Idx: i, Route: r}
		}

		hndlrs := traverse(t.root, normalized)
		if _, exists := hndlrs[r.Method]; exists {
			return nil, Error{Type: ErrTypeDuplicateHandler, Idx: i, Route: r}
		}

		h, numHandlerParams, err := funcs.Convert(r.Handler)
		if err != nil {
			return nil, Error{Type: ErrTypeConversionFailure, Idx: i, Route: r, cause: err}
		} else if numHandlerParams != 0 && numPathParams != numHandlerParams {
			return nil, Error{Type: ErrTypeParamCountMismatch, Idx: i, Route: r}
		}

		if r.Middleware != nil {
			h = applyMiddleware(h, r.Middleware)
		}

		hndlrs[r.Method] = h
	}

	return t, nil
}

func traverse(node *node, path string) map[string]funcs.Handler {
	pathIdx := 0

	child := node.get(path[pathIdx])
	for child != nil {

		// find point where label and path diverge (or one ends)
		labelIdx := 0
		for pathIdx < len(path) && labelIdx < len(child.label) && path[pathIdx] == child.label[labelIdx] {
			pathIdx++
			labelIdx++
		}

		// label has finished
		if labelIdx == len(child.label) {
			node = child
			if pathIdx == len(path) { // label and path are coincident -- probably multiple methods
				break
			}
			child = node.get(path[pathIdx])
			continue
		}

		// if pathIdx is the end of the path, this is a prefix -- split the label and insert
		if pathIdx == len(path) {
			newChild := newNode(child.label[:labelIdx])
			child.label = child.label[labelIdx:]

			newChild.add(child)
			node.add(newChild)

			return newChild.hndlrs
		}

		// path is different from label in middle of label -- split
		newN := newNode(path[pathIdx:])
		branch := newNode(child.label[:labelIdx])
		child.label = child.label[labelIdx:]

		branch.add(child)
		branch.add(newN)
		node.add(branch)

		return newN.hndlrs
	}

	if pathIdx == len(path) {
		return node.hndlrs
	}

	// we've still got path to consume -- add a new child
	ch := newNode(path[pathIdx:])
	node.add(ch)
	return ch.hndlrs
}

type node struct {
	// index[i] == children[i].label[0] always
	index    []byte
	children []*node
	label    string
	hndlrs   map[string]funcs.Handler
}

func newNode(label string) *node {
	return &node{
		hndlrs: make(map[string]funcs.Handler),
		label:  label,
	}
}

func (n *node) add(n2 *node) {
	for i, c := range n.index {
		if c == n2.label[0] {
			n.children[i] = n2
			return
		}
	}

	n.index = append(n.index, n2.label[0])
	n.children = append(n.children, n2)
}

func (n *node) get(b byte) *node {
	for i, ib := range n.index {
		if ib == b {
			return n.children[i]
		}
	}
	return nil
}

func applyMiddleware(h funcs.Handler, mw Middleware) funcs.Handler {
	return func(w http.ResponseWriter, r *http.Request, pathVars funcs.PathVars) {
		mw.Handle(w, r, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h(w, r, pathVars)
		}))
	}
}

// Table manages the routing table and a default handler
type Table struct {
	Default http.Handler
	root    *node
}

func (t *Table) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var variables funcs.PathVars
	_, handlers := t.matchPath(r.RequestURI, variables[:])
	switch {
	case handlers[r.Method] != nil:
		handlers[r.Method](w, r, variables)
	case handlers[MethodAny] != nil:
		handlers[MethodAny](w, r, variables)
	default:
		t.Default.ServeHTTP(w, r)
	}
}

// Vars reparses the request URI and returns any matched variables and whether or not there was a route matched.
func (t *Table) Vars(r *http.Request) ([]string, bool) {
	var variables funcs.PathVars
	i, h  := t.matchPath(r.RequestURI, variables[:])
	return variables[:i], h != nil
}

func (t *Table) matchPath(path string, vars []string) (int, map[string]funcs.Handler) {
	var (
		node            = t.root
		pathIdx, varIdx int
	)
	for {
		child := node.get(path[pathIdx])
		if child == nil {
			return varIdx, nil
		}

		lblIdx := 0
		for {
			switch {
			case path[pathIdx] == child.label[lblIdx]:
				pathIdx++
				lblIdx++
			case child.label[lblIdx] == '*':
				wcStart := pathIdx
				for pathIdx < len(path) && path[pathIdx] != '/' {
					pathIdx++
				}
				vars[varIdx] = path[wcStart:pathIdx]
				varIdx++
				lblIdx++
			default:
				return varIdx, nil
			}

			if pathIdx != len(path) {
				if lblIdx != len(child.label) {
					continue
				}
				node = child
				break
			}

			// path done
			if lblIdx != len(child.label) {
				return varIdx, nil
			}

			// both done
			return varIdx, child.hndlrs
		}
	}
}
