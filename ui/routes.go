package ui

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"servant/storage"
)

type ListTemplateData struct {
	Title           string
	RenderTimestamp int64
}

func rootHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/list", http.StatusFound)
	}
}

func sendDatabase(
	slogLogger *slog.Logger,
	db *storage.Database,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonData, err := db.Serialize()
		if err != nil {
			slogLogger.Error("Failed to serialize database", "error", err)
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
			slogLogger.Error("Failed to send json database data", "error", err)
			return
		}
	}
}

//  TODO: Должен быть какой то значок в статусе джобы
// говорящий о том, выполнилась ли она последний раз или нет

//  TODO: Темная тема
//  TODO: Обновление информации периодическое

func listHandler(
	slogLogger *slog.Logger,
) http.HandlerFunc {
	templateName := "list.html"

	tmpl, fallbackHandler := getTemplateAndFallback(slogLogger, templateName)
	if tmpl == nil {
		return fallbackHandler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		templateData := ListTemplateData{
			Title:           "servant",
			RenderTimestamp: time.Now().Unix(),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		err := tmpl.ExecuteTemplate(w, templateName, templateData)
		if err != nil {
			slogLogger.Error("Failed to execute template", "error", err)
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
	slogLogger *slog.Logger,
	templateName string,
) (*template.Template, func(w http.ResponseWriter, r *http.Request)) {
	templatePath := filepath.Join(templatesDir, templateName)
	tmpl, err := template.New(templateName).
		Funcs(template.FuncMap{}).
		ParseFiles(templatePath)
	if err != nil {
		slogLogger.Error("Failed to parse template", "error", err)
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
