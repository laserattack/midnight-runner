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

	m := createMiddlewaresChain(
		redirectToListMiddleware,
	)

	//  NOTE: Register routes
	mux.Handle("/", m(rootHandler()))
	mux.Handle("/list", m(listHandler(slogLogger, db)))

	return &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
}
