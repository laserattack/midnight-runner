// Package gui: web ui
package gui

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"midnight-runner/storage"
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
	logger *slog.Logger,
	db *storage.Database,
) *http.Server {
	mux := http.NewServeMux()

	m := createMiddlewaresChain(
		logReqMiddleware(logger),
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
	mux.Handle("/list", m(listHandler(logger)))
	mux.Handle("/api/get_database", m(sendDatabase(logger, db)))
	mux.Handle("/api/change_job", m(changeJob(logger, db)))
	mux.Handle("/api/delete_job", m(deleteJob(logger, db)))
	mux.Handle("/api/toggle_job", m(toggleJob(logger, db)))

	return &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
}
