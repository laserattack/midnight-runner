// Package ui: web ui
package ui

import (
	"log/slog"
	"net/http"

	"servant/storage"
)

const (
	templatesDir = "./ui/resources/templates"
	staticDir    = "./ui/resources/static"
)

func CreateWebServer(
	port string,
	slogLogger *slog.Logger,
	db *storage.Database,
) *http.Server {
	mux := http.NewServeMux()

	m := createMiddlewaresChain(
		getLogMiddleware(slogLogger),
	)

	//  NOTE: Register routes

	mux.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir(staticDir)),
		),
	)
	mux.Handle("/", m(rootHandler()))
	mux.Handle("/list", m(listHandler(slogLogger)))
	mux.Handle("/get_database", m(sendDatabase(slogLogger, db)))

	return &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
}
