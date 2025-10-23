// Package ui: web ui
package ui

import (
	"log/slog"
	"net/http"

	"servant/storage"
)

func CreateWebServer(
	port string,
	slogLogger *slog.Logger,
	db *storage.Database,
) *http.Server {
	mux := http.NewServeMux()

	//  NOTE: Register routes

	m := createMiddlewaresChain(
		getLogMiddleware(slogLogger),
	)

	mux.Handle("/", m(rootHandler()))
	mux.Handle("/list", m(listHandler(slogLogger, db)))

	return &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
}
