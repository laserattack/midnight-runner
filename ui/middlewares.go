package ui

import (
	"net/http"
)

type middleware func(http.Handler) http.Handler

func createMiddlewaresChain(ms ...middleware) middleware {
	return func(final http.Handler) http.Handler {
		for i := len(ms) - 1; i >= 0; i-- {
			final = ms[i](final)
		}
		return final
	}
}

func redirectToListMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validRoutes[r.URL.Path] {
			http.Redirect(w, r, "/list", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}
