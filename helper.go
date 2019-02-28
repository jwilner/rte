package rte

import "net/http"

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
		if r.Method == MethodAll {
			defaultSeen[r.Path] = true
		}
	}

	var copied []Route
	for _, r := range routes {
		if !defaultSeen[r.Path] {
			copied = append(copied, r, Route{
				Method:  MethodAll,
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

func GlobalMiddleware(mw Middleware, routes []Route) []Route {
	var copied []Route
	for _, r := range routes {
		r.Middleware = composeMiddleware(r.Middleware, mw)
		copied = append(copied, r)
	}
	return copied
}

func composeMiddleware(mw1, mw2 Middleware) Middleware {
	if mw1 == nil {
		return mw2
	}
	if mw2 == nil {
		return mw1
	}
	return MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		mw1.Handle(w, r, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mw2.Handle(w, r, next)
		}))
	})
}
