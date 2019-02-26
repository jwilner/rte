package rte

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
