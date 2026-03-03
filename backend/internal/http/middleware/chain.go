package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

func Chain(mw ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		if len(mw) == 0 {
			return final
		}

		wrapped := final
		for i := len(mw) - 1; i >= 0; i-- {
			wrapped = mw[i](wrapped)
		}
		return wrapped
	}
}
