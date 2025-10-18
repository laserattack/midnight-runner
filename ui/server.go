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
				fmt.Fprintf(w, "Hello World!\n")
				fmt.Fprintf(w, "Job Scheduler is running...\n")
			}),
	}
}
