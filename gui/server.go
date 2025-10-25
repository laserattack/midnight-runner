// Package gui: web ui
package gui

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"

	"servant/storage"
)

//go:embed resources/*
var resourcesFS embed.FS

var (
	staticFS    fs.FS
	templatesFS fs.FS
)

func init() {
	var err error

	staticFS, err = fs.Sub(resourcesFS, "resources/static")
	if err != nil {
		panic(fmt.Sprintf("failed to create static FS: %v", err))
	}

	templatesFS, err = fs.Sub(resourcesFS, "resources/templates")
	if err != nil {
		panic(fmt.Sprintf("failed to create templates FS: %v", err))
	}
}

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
