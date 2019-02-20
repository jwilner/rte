package rte

import (
	"net/http"

	"strconv"
)

// generated handler wrappers which avoid allocs
// do not edit this file!

// FuncS1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 string
func FuncS1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		s0 string,
	),
) Route {
	return Bind(method, path, funcS1(f))
}

type funcS1 func(
	w http.ResponseWriter,
	r *http.Request,
	s0 string,
)

func (f funcS1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 1 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [1]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		f(
			w,
			r,
			segs[0],
		)
	}, nil
}

// FuncI1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 base-10, max-64 bit integer
func FuncI1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		i0 int64,
	),
) Route {
	return Bind(method, path, funcI1(f))
}

type funcI1 func(
	w http.ResponseWriter,
	r *http.Request,
	i0 int64,
)

func (f funcI1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 1 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [1]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		i0, err := strconv.ParseInt(segs[0], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			i0,
		)
	}, nil
}

// FuncH1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 hex, max-64 bit integer
func FuncH1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		h0 int64,
	),
) Route {
	return Bind(method, path, funcH1(f))
}

type funcH1 func(
	w http.ResponseWriter,
	r *http.Request,
	h0 int64,
)

func (f funcH1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 1 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [1]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		h0, err := strconv.ParseInt(segs[0], 16, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			h0,
		)
	}, nil
}

// FuncU1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 base-10, max-64 bit unsigned integer
func FuncU1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		u0 uint64,
	),
) Route {
	return Bind(method, path, funcU1(f))
}

type funcU1 func(
	w http.ResponseWriter,
	r *http.Request,
	u0 uint64,
)

func (f funcU1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 1 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [1]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		u0, err := strconv.ParseUint(segs[0], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			u0,
		)
	}, nil
}

// FuncS2 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 2 strings
func FuncS2(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		s0, s1 string,
	),
) Route {
	return Bind(method, path, funcS2(f))
}

type funcS2 func(
	w http.ResponseWriter,
	r *http.Request,
	s0, s1 string,
)

func (f funcS2) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		f(
			w,
			r,
			segs[0],

			segs[1],
		)
	}, nil
}

// FuncS1I1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 string
// - 1 base-10, max-64 bit integer
func FuncS1I1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		s0 string,

		i1 int64,
	),
) Route {
	return Bind(method, path, funcS1I1(f))
}

type funcS1I1 func(
	w http.ResponseWriter,
	r *http.Request,
	s0 string,
	i1 int64,
)

func (f funcS1I1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		i1, err := strconv.ParseInt(segs[1], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			segs[0],

			i1,
		)
	}, nil
}

// FuncS1H1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 string
// - 1 hex, max-64 bit integer
func FuncS1H1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		s0 string,

		h1 int64,
	),
) Route {
	return Bind(method, path, funcS1H1(f))
}

type funcS1H1 func(
	w http.ResponseWriter,
	r *http.Request,
	s0 string,
	h1 int64,
)

func (f funcS1H1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		h1, err := strconv.ParseInt(segs[1], 16, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			segs[0],

			h1,
		)
	}, nil
}

// FuncS1U1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 string
// - 1 base-10, max-64 bit unsigned integer
func FuncS1U1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		s0 string,

		u1 uint64,
	),
) Route {
	return Bind(method, path, funcS1U1(f))
}

type funcS1U1 func(
	w http.ResponseWriter,
	r *http.Request,
	s0 string,
	u1 uint64,
)

func (f funcS1U1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		u1, err := strconv.ParseUint(segs[1], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			segs[0],

			u1,
		)
	}, nil
}

// FuncI1S1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 base-10, max-64 bit integer
// - 1 string
func FuncI1S1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		i0 int64,

		s1 string,
	),
) Route {
	return Bind(method, path, funcI1S1(f))
}

type funcI1S1 func(
	w http.ResponseWriter,
	r *http.Request,
	i0 int64,
	s1 string,
)

func (f funcI1S1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		i0, err := strconv.ParseInt(segs[0], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			i0,
			segs[1],
		)
	}, nil
}

// FuncI2 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 2 base-10, max-64 bit integers
func FuncI2(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		i0, i1 int64,
	),
) Route {
	return Bind(method, path, funcI2(f))
}

type funcI2 func(
	w http.ResponseWriter,
	r *http.Request,
	i0, i1 int64,
)

func (f funcI2) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		i0, err := strconv.ParseInt(segs[0], 10, 64)
		if err != nil {
			panic(err)
		}

		i1, err := strconv.ParseInt(segs[1], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			i0,
			i1,
		)
	}, nil
}

// FuncI1H1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 2 base-10, max-64 bit integers
func FuncI1H1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		i0, h1 int64,
	),
) Route {
	return Bind(method, path, funcI1H1(f))
}

type funcI1H1 func(
	w http.ResponseWriter,
	r *http.Request,
	i0, h1 int64,
)

func (f funcI1H1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		i0, err := strconv.ParseInt(segs[0], 10, 64)
		if err != nil {
			panic(err)
		}

		h1, err := strconv.ParseInt(segs[1], 16, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			i0,
			h1,
		)
	}, nil
}

// FuncI1U1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 base-10, max-64 bit integer
// - 1 base-10, max-64 bit unsigned integer
func FuncI1U1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		i0 int64,

		u1 uint64,
	),
) Route {
	return Bind(method, path, funcI1U1(f))
}

type funcI1U1 func(
	w http.ResponseWriter,
	r *http.Request,
	i0 int64,
	u1 uint64,
)

func (f funcI1U1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		i0, err := strconv.ParseInt(segs[0], 10, 64)
		if err != nil {
			panic(err)
		}

		u1, err := strconv.ParseUint(segs[1], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			i0,
			u1,
		)
	}, nil
}

// FuncH1S1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 hex, max-64 bit integer
// - 1 string
func FuncH1S1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		h0 int64,

		s1 string,
	),
) Route {
	return Bind(method, path, funcH1S1(f))
}

type funcH1S1 func(
	w http.ResponseWriter,
	r *http.Request,
	h0 int64,
	s1 string,
)

func (f funcH1S1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		h0, err := strconv.ParseInt(segs[0], 16, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			h0,
			segs[1],
		)
	}, nil
}

// FuncH1I1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 2 hex, max-64 bit integers
func FuncH1I1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		h0, i1 int64,
	),
) Route {
	return Bind(method, path, funcH1I1(f))
}

type funcH1I1 func(
	w http.ResponseWriter,
	r *http.Request,
	h0, i1 int64,
)

func (f funcH1I1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		h0, err := strconv.ParseInt(segs[0], 16, 64)
		if err != nil {
			panic(err)
		}

		i1, err := strconv.ParseInt(segs[1], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			h0,
			i1,
		)
	}, nil
}

// FuncH2 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 2 hex, max-64 bit integers
func FuncH2(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		h0, h1 int64,
	),
) Route {
	return Bind(method, path, funcH2(f))
}

type funcH2 func(
	w http.ResponseWriter,
	r *http.Request,
	h0, h1 int64,
)

func (f funcH2) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		h0, err := strconv.ParseInt(segs[0], 16, 64)
		if err != nil {
			panic(err)
		}

		h1, err := strconv.ParseInt(segs[1], 16, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			h0,
			h1,
		)
	}, nil
}

// FuncH1U1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 hex, max-64 bit integer
// - 1 base-10, max-64 bit unsigned integer
func FuncH1U1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		h0 int64,

		u1 uint64,
	),
) Route {
	return Bind(method, path, funcH1U1(f))
}

type funcH1U1 func(
	w http.ResponseWriter,
	r *http.Request,
	h0 int64,
	u1 uint64,
)

func (f funcH1U1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		h0, err := strconv.ParseInt(segs[0], 16, 64)
		if err != nil {
			panic(err)
		}

		u1, err := strconv.ParseUint(segs[1], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			h0,
			u1,
		)
	}, nil
}

// FuncU1S1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 base-10, max-64 bit unsigned integer
// - 1 string
func FuncU1S1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		u0 uint64,

		s1 string,
	),
) Route {
	return Bind(method, path, funcU1S1(f))
}

type funcU1S1 func(
	w http.ResponseWriter,
	r *http.Request,
	u0 uint64,
	s1 string,
)

func (f funcU1S1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		u0, err := strconv.ParseUint(segs[0], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			u0,
			segs[1],
		)
	}, nil
}

// FuncU1I1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 base-10, max-64 bit unsigned integer
// - 1 base-10, max-64 bit integer
func FuncU1I1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		u0 uint64,

		i1 int64,
	),
) Route {
	return Bind(method, path, funcU1I1(f))
}

type funcU1I1 func(
	w http.ResponseWriter,
	r *http.Request,
	u0 uint64,
	i1 int64,
)

func (f funcU1I1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		u0, err := strconv.ParseUint(segs[0], 10, 64)
		if err != nil {
			panic(err)
		}

		i1, err := strconv.ParseInt(segs[1], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			u0,
			i1,
		)
	}, nil
}

// FuncU1H1 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 1 base-10, max-64 bit unsigned integer
// - 1 hex, max-64 bit integer
func FuncU1H1(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		u0 uint64,

		h1 int64,
	),
) Route {
	return Bind(method, path, funcU1H1(f))
}

type funcU1H1 func(
	w http.ResponseWriter,
	r *http.Request,
	u0 uint64,
	h1 int64,
)

func (f funcU1H1) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		u0, err := strconv.ParseUint(segs[0], 10, 64)
		if err != nil {
			panic(err)
		}

		h1, err := strconv.ParseInt(segs[1], 16, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			u0,
			h1,
		)
	}, nil
}

// FuncU2 creates a route which matches the supplied method and path. In addition to a response writer, and
// a request object, the provided handler requires the matched path contain in order:
// - 2 base-10, max-64 bit unsigned integers
func FuncU2(
	method,
	path string,
	f func(
		w http.ResponseWriter,
		r *http.Request,
		u0, u1 uint64,
	),
) Route {
	return Bind(method, path, funcU2(f))
}

type funcU2 func(
	w http.ResponseWriter,
	r *http.Request,
	u0, u1 uint64,
)

func (f funcU2) Bind(segIdxes []int) (http.HandlerFunc, error) {
	if len(segIdxes) != 2 {
		return nil, ErrWrongNumParams
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var segs [2]string
		findNSegments(r.URL.Path, segIdxes[:], segs[:])

		u0, err := strconv.ParseUint(segs[0], 10, 64)
		if err != nil {
			panic(err)
		}

		u1, err := strconv.ParseUint(segs[1], 10, 64)
		if err != nil {
			panic(err)
		}

		f(
			w,
			r,
			u0,
			u1,
		)
	}, nil
}
