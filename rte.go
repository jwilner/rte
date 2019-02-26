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
	// MethodNotAllowed can be provided used as a method within a route to handle scenarios when the path but not
	// the method are matched.
	//
	// E.g., serve gets on '/foo/:foo_id' and return a 405 for everything else (405 handler can also access path vars):
	// 		_ = rte.Must(rte.Routes(
	// 			"GET /foo/:foo_id", handlerGet,
	// 			rte.MethodNotAllowed + " /foo/:foo_id", handler405,
	// 		))
	MethodNotAllowed = "405"
)

const (
	wildcard, wildcardSlash = "*", "*/"
)

var regexpReqLine = regexp.MustCompile(`^(\S+)\s+(.+)$`)

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

// Routes is a vanity constructor for constructing literal routing tables.
func Routes(is ...interface{}) []Route {
	var routes []Route

	idxReqLine := 0
	for idxReqLine < len(is) {
		if is[idxReqLine] == nil {
			idxReqLine++
			continue
		}

		if rs, ok := is[idxReqLine].([]Route); ok {
			routes = append(routes, rs...)
			idxReqLine++
			continue
		}

		r := Route{}
		{
			str, ok := is[idxReqLine].(string)
			if !ok {
				panic(fmt.Sprintf(
					"rte.Routes: argument %d must be either a string or a []Route but got %v",
					idxReqLine,
					is[idxReqLine],
				))
			}

			match := regexpReqLine.FindStringSubmatch(str)
			if len(match) == 0 {
				panic(fmt.Sprintf("rte.Routes: argument %d must match %q but got %q", idxReqLine, regexpReqLine, str))
			}
			r.Method, r.Path = match[1], match[2]
		}

		idxHandler, idxMW := idxReqLine+1, idxReqLine+2
		if idxHandler >= len(is) {
			panic(fmt.Sprintf(
				"rte.Routes: missing a handler for \"%v %v\" at argument %d",
				r.Method,
				r.Path,
				idxHandler,
			))
		}

		if _, _, err := funcs.Convert(is[idxHandler]); err != nil {
			panic(fmt.Sprintf(
				"rte.Routes: invalid handler for \"%v %v\" in position %v: %v",
				r.Method,
				r.Path,
				idxHandler,
				err,
			))
		}
		r.Handler = is[idxHandler]

		if idxMW < len(is) {
			if mw, ok := is[idxMW].(Middleware); ok {
				r.Middleware = mw
				routes = append(routes, r)
				idxReqLine = idxMW + 1
				continue
			}
		}

		idxReqLine = idxHandler + 1
		routes = append(routes, r)
	}

	return routes
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
		numPathParams := 0
		for _, seg := range strings.SplitAfter(r.Path, "/")[1:] {
			// normalize
			seg, err := normalize(seg)
			if err != nil {
				return nil, Error{Type: ErrTypeInvalidSegment, Idx: i, Route: r, cause: err}
			} else if seg == wildcard || seg == wildcardSlash {
				numPathParams++
			}

			if n.children[seg] == nil {
				n.children[seg] = newNode()
			}

			n = n.children[seg]
		}

		if _, has := n.methods[r.Method]; has {
			return nil, Error{Type: ErrTypeDuplicateHandler, Idx: i, Route: r}
		}

		h, numHandlerParams, err := funcs.Convert(r.Handler)
		if err != nil {
			return nil, Error{Type: ErrTypeConversionFailure, Idx: i, Route: r, cause: err}
		} else if numPathParams != numHandlerParams {
			return nil, Error{Type: ErrTypeParamCountMismatch, Idx: i, Route: r}
		}

		if r.Middleware != nil {
			h = applyMiddleware(h, r.Middleware)
		}

		n.methods[r.Method] = h
	}

	t.Default = http.NotFoundHandler()

	return t, nil
}

func newNode() *node {
	return &node{children: make(map[string]*node), methods: make(map[string]funcs.Handler)}
}

func applyMiddleware(h funcs.Handler, mw Middleware) funcs.Handler {
	return func(w http.ResponseWriter, r *http.Request, pathVars funcs.PathVars) {
		mw.Handle(w, r, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h(w, r, pathVars)
		}))
	}
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
		return wildcardSlash, nil
	default:
		return wildcard, nil
	}
}

// Table manages the routing table and a default handler
type Table struct {
	Default http.Handler
	root    *node
}

func (t *Table) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		i      int
		params funcs.PathVars
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
	methods  map[string]funcs.Handler
}

func (n *node) match(seg string) (string, *node) {
	if c := n.children[seg]; c != nil {
		return "", c
	} else if l := len(seg) - 1; l >= 0 && seg[l] == '/' {
		return seg[:l], n.children[wildcardSlash]
	} else {
		return seg, n.children[wildcard]
	}
}
