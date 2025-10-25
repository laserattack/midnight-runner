// Package gui: web ui
package gui

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"servant/storage"
)

//go:embed resources/*
var resourcesFS embed.FS

var (
	// fs.Sub returns an error only if the 2nd argument
	// is a syntactically invalid path. Since these
	// are valid string literals, error handling is unnecessary
	staticFS, _    = fs.Sub(resourcesFS, "resources/static")
	templatesFS, _ = fs.Sub(resourcesFS, "resources/templates")
)

func CreateWebServer(
	port string,
	slogLogger *slog.Logger,
	db *storage.Database,
) *http.Server {
	mux := http.NewServeMux()

	m := createMiddlewaresChain(
		logReqMiddleware(slogLogger),
	)

	//  NOTE: Register routes

	mux.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.FS(staticFS)),
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
