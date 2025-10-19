// Package ui: web ui
package ui

import (
	"html/template"
	"log/slog"
	"net/http"

	"servant/storage"
)

type TemplateData struct {
	Title string
	Jobs  storage.Jobs
}

func CreateWebServer(
	port string,
	slogLogger *slog.Logger,
	db *storage.Database,
) *http.Server {
	mux := http.NewServeMux()

	m := createMiddlewaresChain(
		redirectToListMiddleware,
	)

	mux.Handle("/", m(rootHandler()))
	mux.Handle("/list", m(listHandler(slogLogger, db)))

	return &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
}

func rootHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/list", http.StatusFound)
	})
}

func listHandler(
	slogLogger *slog.Logger,
	db *storage.Database,
) http.HandlerFunc {
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1>{{.Title}}</h1>
    
    {{if .Jobs}}
        <p>All jobs count: {{len .Jobs}}</p>
        <table border="1">
            <thead>
                <tr>
                    <th>Name</th>
                    <th>Type</th>
                    <th>Description</th>
                    <th>Command</th>
                    <th>Cron</th>
                    <th>Status</th>
                    <th>Timeout</th>
                    <th>Max retries</th>
                    <th>Retry interval</th>
                    <th>Updated at</th>
                </tr>
            </thead>
            <tbody>
                {{range $name, $job := .Jobs}}
                <tr>
                    <td>{{$name}}</td>
                    <td>{{$job.Type}}</td>
                    <td>{{$job.Description}}</td>
                    <td>{{$job.Config.Command}}</td>
                    <td>{{$job.Config.CronExpression}}</td>
                    <td>{{$job.Config.Status}}</td>
                    <td>{{$job.Config.Timeout}}</td>
                    <td>{{$job.Config.MaxRetries}}</td>
                    <td>{{$job.Config.RetryInterval}}</td>
                    <td>{{$job.Metadata.UpdatedAt}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    {{else}}
        <p>No jobs</p>
    {{end}}
</body>
</html>`

	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.New("webpage").
			Funcs(template.FuncMap{}).
			Parse(htmlTemplate)
		if err != nil {
			slogLogger.Error("Failed to parse template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		templateData := TemplateData{
			Title: "Jobs list",
			Jobs:  db.Jobs,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		err = tmpl.Execute(w, templateData)
		if err != nil {
			slogLogger.Error("Failed to execute template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}
