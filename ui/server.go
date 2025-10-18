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
				_, err := fmt.Fprintf(w, "Hello World!\n")
				if err != nil {
					slogLogger.Error("Failed to write response", "error", err)
					return
				}
			}),
	}
}
