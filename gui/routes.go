package gui

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"midnight-runner/storage"
	"midnight-runner/utils"
)

func rootHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/list", http.StatusFound)
	}
}

func lastLog(
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Count uint `json:"count"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Error("Error decode lastLog json data", "error", err)
			return
		}

		defer func() {
			if err = r.Body.Close(); err != nil {
				logger.Error("Failed to close request body", "error", err)
			}
		}()

		type LogEntry struct {
			Time    string         `json:"time"`
			Message string         `json:"message"`
			Attrs   map[string]any `json:"attrs,omitempty"`
		}

		if bh, ok := logger.Handler().(*utils.SlogBufferedHandler); ok {
			records := bh.GetLastRecords(int(req.Count))
			entries := make([]LogEntry, len(records))

			for i, rec := range records {
				attrs := make(map[string]any)
				rec.Attrs(func(a slog.Attr) bool {
					attrs[a.Key] = a.Value.Any()
					return true
				})

				entries[i] = LogEntry{
					Time:    rec.Time.Format(time.RFC3339),
					Message: rec.Message,
					Attrs:   attrs,
				}
			}

			// send

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(entries); err != nil {
				logger.Error("Failed to encode log entries to JSON",
					"error", err,
				)
				return
			}
		} else {
			logger.Error("Logger handler is not a SlogBufferedHandler")
			return
		}
	}
}

func execJob(
	logger *slog.Logger,
	db *storage.Database,
	ctx context.Context,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Error("Error decode execJob json data", "error", err)
			return
		}

		defer func() {
			if err = r.Body.Close(); err != nil {
				logger.Error("Failed to close request body", "error", err)
			}
		}()

		db.ExecJob(req.Name, ctx, logger)
	}
}

func toggleJob(
	logger *slog.Logger,
	db *storage.Database,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Error("Error decode toggleJob json data", "error", err)
			return
		}

		defer func() {
			if err = r.Body.Close(); err != nil {
				logger.Error("Failed to close request body", "error", err)
			}
		}()

		db.ToggleJob(req.Name)
	}
}

func deleteJob(
	logger *slog.Logger,
	db *storage.Database,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Error("Error decode deleteJob json data", "error", err)
			return
		}

		defer func() {
			if err = r.Body.Close(); err != nil {
				logger.Error("Failed to close request body", "error", err)
			}
		}()

		db.DeleteJob(req.Name)
	}
}

func changeJob(
	logger *slog.Logger,
	db *storage.Database,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name          string `json:"name"`
			Description   string `json:"description"`
			Command       string `json:"command"`
			Cron          string `json:"cron"`
			Timeout       uint   `json:"timeout"`
			MaxRetries    uint   `json:"maxRetries"`
			RetryInterval uint   `json:"retryInterval"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Error("Error decode changeJob json data", "error", err)
			return
		}

		defer func() {
			if err = r.Body.Close(); err != nil {
				logger.Error("Failed to close request body", "error", err)
			}
		}()

		j := storage.ShellJob(
			req.Description,
			req.Command,
			req.Cron,
			int(req.Timeout),
			int(req.MaxRetries),
			int(req.RetryInterval),
		)

		db.SetJob(j, req.Name)
	}
}

func sendDatabase(
	logger *slog.Logger,
	db *storage.Database,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonData, err := db.SerializeWithLock()
		if err != nil {
			logger.Error("Failed to serialize database", "error", err)
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

func listHandler(
	logger *slog.Logger,
) http.HandlerFunc {
	templateName := "list.html"

	tmpl, fallbackHandler := getTemplateAndFallback(logger, templateName)
	if tmpl == nil {
		return fallbackHandler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		templateData := struct {
			Title           string
			RenderTimestamp int64
		}{
			Title:           "🌙⚙️ Midnight Runner",
			RenderTimestamp: time.Now().Unix(),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		err := tmpl.ExecuteTemplate(w, templateName, templateData)
		if err != nil {
			logger.Error("Failed to execute template", "error", err)
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
		return nil, func(w http.ResponseWriter, r *http.Request) {}
	}
	return tmpl, nil
}
