// Package ui: web ui
package ui

import (
	"fmt"
	"log/slog"
	"net/http"
)

func CreateWebServer(port string, slogLogger *slog.Logger) *http.Server {
	return &http.Server{
		Addr: ":" + port,
		Handler: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprintf(w, "Hello World!\n")
				_, _ = fmt.Fprintf(w, "Job Scheduler is running...\n")
			}),
	}
}
