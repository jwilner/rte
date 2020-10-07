// Package rte provides simple, performant routing.
// - Define individual routes with `rte.Func` and generated siblings
// - Combine them into a table with `rte.Must` or `rte.New`
package rte

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/jwilner/rte/internal/funcs"
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
	ErrTypeMethodEmpty = iota + 1
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
	// ErrTypeConflictingRoutes is returned when a route would be obscured by a wildcard.
	ErrTypeConflictingRoutes
)

// TableError encapsulates table construction errors
type TableError struct {
	Type, Idx int
	Route     Route
	Msg       string
}

func (e *TableError) Error() string {
	return fmt.Sprintf("route %d %q: %v", e.Idx, e.Route, e.Msg)
}

// Must builds routes into a Table and panics if there's an error
func Must(routes []Route) *Table {
	t, e := New(routes)
	if e != nil {
		panic(e.Error())
	}
	return t
}

var (
	regexpNormalize  = regexp.MustCompile(`:[^/]*`)
	regexpInvalidVar = regexp.MustCompile(`[^/]:`)
)

// New builds routes into a Table or returns an error
func New(routes []Route) (*Table, error) {
	t := &Table{
		root:    newNode("", 0),
		Default: http.NotFoundHandler(),
	}

	seenMethods := map[string]bool{}
	maxVars := len(funcs.PathVars{})
	for i, r := range routes {
		if r.Method == "" {
			return nil, &TableError{Type: ErrTypeMethodEmpty, Idx: i, Route: r, Msg: "method cannot be empty"}
		}

		if r.Handler == nil {
			return nil, &TableError{Type: ErrTypeNilHandler, Idx: i, Route: r, Msg: "handler cannot be nil"}
		}

		if r.Path == "" {
			return nil, &TableError{Type: ErrTypePathEmpty, Idx: i, Route: r, Msg: "path cannot be empty"}
		}

		if r.Path[0] != '/' {
			return nil, &TableError{Type: ErrTypeNoInitialSlash, Idx: i, Route: r, Msg: "no initial slash"}
		}

		if strings.Contains(r.Path, "*") || regexpInvalidVar.MatchString(r.Path) {
			return nil, &TableError{Type: ErrTypeInvalidSegment, Idx: i, Route: r, Msg: "invalid segment"}
		}

		var numPathParams int
		for _, c := range r.Path {
			if c == ':' {
				numPathParams++
			}
		}

		if numPathParams > maxVars {
			return nil, &TableError{
				Type:  ErrTypeOutOfRange,
				Idx:   i,
				Route: r,
				Msg:   fmt.Sprintf("path has more than %v parameters", maxVars),
			}
		}

		h, numHandlerParams, ok := funcs.Convert(r.Handler)
		if !ok {
			return nil, &TableError{
				Type:  ErrTypeConversionFailure,
				Idx:   i,
				Route: r,
				Msg:   fmt.Sprintf("handler has an unsupported signature: %T", r.Handler),
			}
		} else if numHandlerParams != 0 && numPathParams != numHandlerParams {
			return nil, &TableError{
				Type:  ErrTypeParamCountMismatch,
				Idx:   i,
				Route: r,
				Msg:   "path and handler have different numbers of parameters",
			}
		}

		if r.Middleware != nil {
			h = applyMiddleware(h, r.Middleware)
		}

		if !seenMethods[r.Method] {
			seenMethods[r.Method] = true
			t.methods = append(t.methods, r.Method)
			if r.Method == MethodAny {
				// we'll want to always check for MethodAny, too, in our subtrees
				t.methodMask = 1 << uint(len(t.methods)-1)
			}
		}

		var methodFlag uint
		for i, m := range t.methods {
			if m == r.Method {
				methodFlag = 1 << uint(i)
			}
		}

		normalized := regexpNormalize.ReplaceAllString(r.Path, "*")
		if err := insert(t.root, methodFlag, r.Method, normalized, h); err != nil {
			err.Route = r
			err.Idx = i
			return nil, err
		}
	}

	return t, nil
}

func insert(node *node, methodFlag uint, method, path string, h funcs.Handler) *TableError {
	node.methods |= methodFlag // mark this node as containing our current method

	pathIdx := 0

	child := node.child(path[pathIdx])
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
			node.methods |= methodFlag // mark this new node as containing our current method

			if pathIdx == len(path) { // label and path are coincident -- probably multiple methods
				break
			}
			child = node.child(path[pathIdx])
			continue
		}

		// if pathIdx is the end of the path, this is a prefix -- split the label and insert
		if pathIdx == len(path) {
			// note that the order in which nodes are added is significant here, because we're about to
			// mutate labels and that's what things are internally keyed by -- always add parents first.
			newChild := newNode(child.label[:labelIdx], methodFlag|child.methods)
			newChild.setHandler(method, h)
			node.addChild(newChild)

			child.label = child.label[labelIdx:]
			newChild.addChild(child)
			return nil // no conflict possible
		}

		// path is different from label in middle of label -- split

		// note that the order in which nodes are added is significant here, because we're about to
		// mutate labels and that's what things are internally keyed by -- always add parents first.
		branch := newNode(child.label[:labelIdx], child.methods|methodFlag)
		node.addChild(branch)

		newN := newNode(path[pathIdx:], methodFlag)
		newN.setHandler(method, h)

		child.label = child.label[labelIdx:]

		branch.addChild(newN) // error is impossible b/c we know branch has no children
		branch.addChild(child)

		return checkConflict(path[:pathIdx], branch)
	}

	if pathIdx == len(path) {
		if node.handler(method) != nil {
			return &TableError{Type: ErrTypeDuplicateHandler, Msg: "duplicate handler"}
		}
		node.setHandler(method, h)
		return nil
	}

	// we've still got path to consume -- add a new child
	ch := newNode(path[pathIdx:], methodFlag)
	ch.setHandler(method, h)
	node.addChild(ch)

	return checkConflict(path[:pathIdx], node)
}

type node struct {
	// methods is a bit mask represent the different HTTP methods available in this subtree
	methods  uint
	children []*node
	label    string
	hndlrs   []methodHandler
}

func newNode(label string, methodFlags uint) *node {
	return &node{label: label, methods: methodFlags}
}

func (n *node) addChild(n2 *node) {
	for i, c := range n.children {
		if c.label[0] == n2.label[0] {
			n.children[i] = n2
			return
		}
	}

	l := len(n.children)
	// micro optimization! always resize to exactly fit one more. arguably not worth it.
	// trades marginally slower init for marginally smaller memory footprint
	newC := make([]*node, l+1)
	copy(newC, n.children)
	newC[l] = n2
	n.children = newC
}

func (n *node) child(b byte) *node {
	for _, c := range n.children {
		if c.label[0] == b {
			return c
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
	Default    http.Handler
	root       *node
	methods    []string
	methodMask uint
}

func (t *Table) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	methods := t.acceptMethods(r)
	if methods == 0 {
		t.Default.ServeHTTP(w, r)
		return
	}

	var variables funcs.PathVars
	if _, node := t.matchPath(methods, r.RequestURI, variables[:]); node != nil {
		if h := node.handler(r.Method); h != nil {
			h(w, r, variables)
			return
		}
		if h := node.handler(MethodAny); h != nil {
			h(w, r, variables)
			return
		}
	}

	t.Default.ServeHTTP(w, r)
}

type methodHandler struct {
	Method  string
	Handler funcs.Handler
}

func (n *node) handler(m string) funcs.Handler {
	for _, v := range n.hndlrs {
		if v.Method == m {
			return v.Handler
		}
	}
	return nil
}

func (n *node) setHandler(m string, hndlr funcs.Handler) {
	// micro optimization! always resize to exactly fit one more. arguably not worth it.
	// trades marginally slower init for marginally smaller memory footprint
	l := len(n.hndlrs)
	newH := make([]methodHandler, l+1)
	copy(newH, n.hndlrs)
	newH[l] = methodHandler{m, hndlr}
	n.hndlrs = newH
}

// Vars reparses the request URI and returns any matched variables and whether or not there was a route matched.
func (t *Table) Vars(r *http.Request) ([]string, bool) {
	var variables funcs.PathVars
	i, h := t.matchPath(t.acceptMethods(r), r.RequestURI, variables[:])
	return variables[:i], h != nil
}

func (t *Table) acceptMethods(r *http.Request) uint {
	// don't let MethodAny be used as a request method
	if r.Method == MethodAny {
		return 0
	}

	acceptedMethods := t.methodMask
	for i, m := range t.methods {
		if m == r.Method {
			return acceptedMethods | 1<<uint(i)
		}
	}
	return acceptedMethods
}

func (t *Table) matchPath(methodMask uint, path string, vars []string) (int, *node) {
	var (
		node            = t.root
		pathIdx, varIdx int
	)
	for {
		// is there a non-nil sub-tree matching this path explicitly with our methods in it?
		child := node.child(path[pathIdx])
		if child == nil || (child.methods&methodMask) == 0 {
			// is there a non-nil sub-tree matching this path via a wildcard with our methods in it?
			if child = node.child('*'); child == nil || (child.methods&methodMask) == 0 {
				return varIdx, nil
			}
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
			return varIdx, child
		}
	}
}

// checks whether any routes anchored at the current node are obscured by wildcards
// only matters if methods are the same
func checkConflict(prefix string, n *node) *TableError {
	wildChild := n.child('*')
	if len(n.children) < 2 || wildChild == nil {
		return nil
	}

	var overlap *node
	for _, n := range n.children {
		if n == wildChild {
			continue
		}
		if wildChild.methods&n.methods > 0 {
			overlap = n
			break
		}
	}

	if overlap == nil {
		// both wildcards and static but methods are different
		return nil
	}

	// we've got a conflict; now gather info for the error message

	staticPrefix := make(map[string][]string)
	wildPrefix := make(map[string][]string)

	for _, n := range []struct {
		Map  map[string][]string
		Node *node
	}{
		{wildPrefix, wildChild},
		{staticPrefix, overlap},
	} {
		for _, v := range extract(n.Node) {
			method := v[len(v)-1]
			absPath := strings.Join(append([]string{n.Node.label}, v[:len(v)-1]...), "")
			n.Map[method] = append(n.Map[method], absPath)
		}
	}

	var conflicts []string
	for method := range staticPrefix {
		if wildPrefix[method] != nil {
			for _, s := range append(wildPrefix[method], staticPrefix[method]...) {
				conflicts = append(conflicts, fmt.Sprintf("\"%v %v%v\"", method, prefix, s))
			}
		}
	}

	return &TableError{Type: ErrTypeConflictingRoutes, Msg: "conflicting routes: " + strings.Join(conflicts, ", ")}
}

// enumerates routes from current node, with method at end:
// ["/foo", "bar", "GET"]
func extract(n *node) (sub [][]string) {
	for _, c := range n.children {
		for _, v := range extract(c) {
			sub = append(sub, append([]string{c.label}, v...))
		}
	}
	for _, h := range n.hndlrs {
		sub = append(sub, []string{h.Method})
	}
	return
}
