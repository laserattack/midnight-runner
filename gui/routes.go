package gui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"midnight-runner/storage"
)

func rootHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/list", http.StatusFound)
	}
}

func deleteJob(
	logger *slog.Logger,
	db *storage.Database,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jobData struct {
			Name string `json:"name"`
		}

		err := json.NewDecoder(r.Body).Decode(&jobData)
		if err != nil {
			logger.Error("Error decode deleteJob json data", "error", err)
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		defer func() {
			if err = r.Body.Close(); err != nil {
				logger.Error("Failed to close request body", "error", err)
			}
		}()

		db.DeleteJob(jobData.Name)
	}
}

func changeJob(
	logger *slog.Logger,
	db *storage.Database,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//  TODO: Тут CronExpression должно быть
		var jobData struct {
			Name          string `json:"name"`
			Description   string `json:"description"`
			Command       string `json:"command"`
			Cron          string `json:"cron"`
			Timeout       int    `json:"timeout"`
			MaxRetries    int    `json:"maxRetries"`
			RetryInterval int    `json:"retryInterval"`
		}

		err := json.NewDecoder(r.Body).Decode(&jobData)
		if err != nil {
			logger.Error("Error decode changeJob json data", "error", err)
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		defer func() {
			if err = r.Body.Close(); err != nil {
				logger.Error("Failed to close request body", "error", err)
			}
		}()

		j := storage.ShellJob(
			jobData.Description,
			jobData.Command,
			jobData.Cron,
			jobData.Timeout,
			jobData.MaxRetries,
			jobData.RetryInterval,
		)

		db.SetJob(j, jobData.Name)
	}
}

func sendDatabase(
	logger *slog.Logger,
	db *storage.Database,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonData, err := db.Serialize()
		if err != nil {
			logger.Error("Failed to serialize database", "error", err)
			http.Error(
				w,
				"Failed to serialize database",
				http.StatusInternalServerError,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_, err = w.Write(jsonData)
		if err != nil {
			logger.Error("Failed to send json database data", "error", err)
			return
		}
	}
}

//  TODO: Должен быть какой то значок в статусе джобы
// говорящий о том, выполнилась ли она последний раз или нет

type ListTemplateData struct {
	Title           string
	RenderTimestamp int64
}

func listHandler(
	logger *slog.Logger,
) http.HandlerFunc {
	templateName := "list.html"

	tmpl, fallbackHandler := getTemplateAndFallback(logger, templateName)
	if tmpl == nil {
		return fallbackHandler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		templateData := ListTemplateData{
			Title:           "🌙⚙️ Midnight Runner",
			RenderTimestamp: time.Now().Unix(),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		err := tmpl.ExecuteTemplate(w, templateName, templateData)
		if err != nil {
			logger.Error("Failed to execute template", "error", err)
			http.Error(
				w,
				"Internal Server Error",
				http.StatusInternalServerError,
			)
			return
		}
	}
}

// getTemplateAndFallback loads and parses an HTML template
// from the file system. It returns the parsed template and
// a fallback HTTP handler. If template parsing fails,
// it returns nil for the template and a fallback handler
// that returns an HTTP 500 error with details about the failure.

//  TODO: Тут надо какую то html страничку возвращать красивую
// с описанием ошибки. Хранить ее можно в константной строке
// чтобы ничего не парсить

func getTemplateAndFallback(
	logger *slog.Logger,
	templateName string,
) (parsedTemplate *template.Template, fallbackHandler http.HandlerFunc) {
	tmpl, err := template.New(templateName).
		Funcs(template.FuncMap{}).
		ParseFS(templatesFS, templateName)
	if err != nil {
		logger.Error("Failed to parse template", "error", err)
		return nil, func(w http.ResponseWriter, r *http.Request) {
			http.Error(
				w,
				fmt.Sprintf("Failed to parse template '%s'", templateName),
				http.StatusInternalServerError,
			)
		}
	}
	return tmpl, nil
}
