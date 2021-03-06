package rte

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jwilner/rte/internal/funcs"
)

// Routes is a vanity constructor for constructing literal routing tables. It enforces types at runtime. An invocation
// can be zero or more combinations. Each combination can be one of:
// - nil
// - "METHOD", handler
// - "METHOD PATH", handler
// - "PATH", handler
// - "", handler
// - "METHOD", handler, middleware
// - "METHOD PATH", handler, middleware
// - "PATH", handler, middleware
// - "", handler, middleware
// - Route
// - []Route
// - "PATH", []Route (identical to rte.Prefix("PATH", routes))
// - "PATH", []Route, middleware (identical to rte.Wrap(rte.Prefix("PATH", routes), middleware))
func Routes(is ...interface{}) []Route {
	var routes []Route

	idxReqLine := 0
	for idxReqLine < len(is) {
		if is[idxReqLine] == nil {
			idxReqLine++
			continue
		}

		var reqLine string
		switch v := is[idxReqLine].(type) {
		case Route:
			routes = append(routes, v)
			idxReqLine++
			continue
		case []Route:
			routes = append(routes, v...)
			idxReqLine++
			continue
		case string:
			reqLine = v
		default:
			panic(fmt.Sprintf(
				"rte.Routes: argument %d must be either a string, a Route, or a []Route but got %T: %v",
				idxReqLine,
				is[idxReqLine],
				is[idxReqLine],
			))
		}

		idxHandler := idxReqLine + 1
		if idxHandler >= len(is) {
			panic(fmt.Sprintf("rte.Routes: missing a target for %q at argument %d", reqLine, idxHandler))
		}

		var newRoutes []Route
		switch v := is[idxHandler].(type) {
		case []Route:
			if len(reqLine) == 0 || reqLine[0] != '/' {
				panic("rte.Routes: if providing []Route as a target, reqLine must be a prefix")
			}
			newRoutes = Prefix(reqLine, v)
		default:
			var r Route
			split := strings.SplitN(reqLine, " ", 2)
			switch len(split) {
			case 2:
				r.Method, r.Path = split[0], strings.Trim(split[1], " ")
			case 1:
				if len(split[0]) > 0 && split[0][0] == '/' {
					r.Path = split[0]
				} else {
					r.Method = split[0]
				}
			}
			if _, _, ok := funcs.Convert(v); !ok {
				panic(fmt.Sprintf(
					"rte.Routes: invalid handler for \"%v %v\" in position %v: %T",
					r.Method,
					r.Path,
					idxHandler,
					v,
				))
			}
			r.Handler = v

			newRoutes = []Route{r}
		}

		if idxMW := idxHandler + 1; idxMW < len(is) {
			if mw, ok := is[idxMW].(Middleware); ok {
				routes = append(routes, Wrap(mw, newRoutes)...)
				idxReqLine = idxMW + 1
				continue
			}
		}

		idxReqLine = idxHandler + 1
		routes = append(routes, newRoutes...)
	}

	return routes
}

// OptTrailingSlash ensures that the provided routes will perform the same regardless of whether or not they have a
// trailing slash.
func OptTrailingSlash(routes []Route) []Route {
	const (
		seenNoSlash = 1 << 0
		seenSlash   = 1 << 1
	)

	classify := func(r Route) (uint8, string) {
		k := r.Method + " " + r.Path
		if k[len(k)-1] != '/' {
			return seenNoSlash, k
		}
		return seenSlash, k[:len(k)-1]
	}

	seen := make(map[string]uint8)
	for _, r := range routes {
		t, k := classify(r)
		seen[k] |= t
	}

	added := make(map[string]bool)

	var copied []Route
	for _, r := range routes {
		_, k := classify(r)
		copied = append(copied, r)

		switch seen[k] {
		case seenSlash: // only seen slash, add no slash
			if r.Path == "/" {
				continue
			}

			c := r
			c.Path = r.Path[:len(c.Path)-1]

			_, k2 := classify(r)
			if !added[k2] {
				copied = append(copied, c)
				added[k2] = true
			}

		case seenNoSlash:
			c := r
			c.Path = r.Path + "/"

			_, k2 := classify(r)
			if !added[k2] {
				copied = append(copied, c)
				added[k2] = true
			}
		}
	}

	return copied
}

// Prefix adds the given prefix to all of the contained routes; no verification is performed of e.g. leading slashes
func Prefix(prefix string, routes []Route) []Route {
	var prefixed []Route
	for _, r := range routes {
		r.Path = prefix + r.Path
		prefixed = append(prefixed, r)
	}
	return prefixed
}

// DefaultMethod adds a default method handler to any paths without one.
func DefaultMethod(hndlr interface{}, routes []Route) []Route {
	defaultSeen := make(map[string]bool)
	for _, r := range routes {
		if r.Method == MethodAny {
			defaultSeen[r.Path] = true
		}
	}

	var copied []Route
	for _, r := range routes {
		if !defaultSeen[r.Path] {
			copied = append(copied, r, Route{
				Method:  MethodAny,
				Path:    r.Path,
				Handler: hndlr,
			})
			defaultSeen[r.Path] = true
			continue
		}

		copied = append(copied, r)
	}

	return copied
}

// Wrap registers a middleware across all provide routes. If a middleware is already set, that middleware will be
// invoked second.
func Wrap(mw Middleware, routes []Route) []Route {
	var copied []Route
	for _, r := range routes {
		if r.Middleware != nil {
			r.Middleware = Compose(mw, r.Middleware)
		} else {
			r.Middleware = mw
		}
		copied = append(copied, r)
	}
	return copied
}

// Compose combines one or more middlewares into a single middleware. The composed middleware will proceed left to right
// through the middleware (and exit right to left).
func Compose(mw Middleware, mws ...Middleware) Middleware {
	mws = append([]Middleware{mw}, mws...)
	mw = mws[len(mws)-1]
	for i := len(mws) - 2; i >= 0; i-- {
		mw1, mw2 := mws[i], mw
		mw = MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
			mw1.Handle(w, r, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mw2.Handle(w, r, next)
			}))
		})
	}
	return mw
}

// RecoveryMiddleware returns a middleware which converts any panics into 500 status http errors and stops the panic. If
// a non-nil log is provided, any panic will be logged.
func RecoveryMiddleware(log interface{ Println(...interface{}) }) Middleware {
	if log == nil {
		return MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
			defer func() {
				if p := recover(); p != nil {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
	return MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		defer func() {
			if p := recover(); p != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(p)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
