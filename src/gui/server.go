// Package gui: web ui
package gui

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"cronshroom/storage"
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
	httpLogger *slog.Logger,
	logger *slog.Logger,
	db *storage.Database,
	ctx context.Context,
) *http.Server {
	mux := http.NewServeMux()

	m := createMiddlewaresChain(
		logReqMiddleware(httpLogger),
	)

	// NOTE: Register routes

	mux.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.FS(staticFS)),
		),
	)
	mux.Handle("/", m(rootHandler()))
	mux.Handle("/list", m(listHandler(logger)))

	// NOTE: Api routes

	{
		mux.Handle("/api/get_database", m(sendDatabase(logger, db)))
		mux.Handle("/api/change_job", m(changeJob(logger, db)))
		mux.Handle("/api/delete_job", m(deleteJob(logger, db)))
		mux.Handle("/api/toggle_job", m(toggleJob(logger, db)))
		mux.Handle("/api/exec_job", m(execJob(logger, db, ctx)))
		mux.Handle("/api/last_log", m(lastLog(logger)))
	}

	return &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
}
