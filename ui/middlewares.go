package ui

import (
	"log/slog"
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

func getLogMiddleware(slogLogger *slog.Logger) middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slogLogger.Info("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
			h.ServeHTTP(w, r)
		})
	}
}
