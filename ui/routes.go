package ui

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"servant/storage"
)

const templatesDir = "./ui/resources/"

func rootHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/list", http.StatusFound)
	}
}

//  TODO: Должен быть какой то значок в статусе джобы
// говорящий о том, выполнилась ли она последний раз или нет

//  TODO: Темная тема
//  TODO: Обновление информации периодическое

func listHandler(
	slogLogger *slog.Logger,
	db *storage.Database,
) http.HandlerFunc {
	templateName := "list.html"

	tmpl, err := template.New(templateName).
		Funcs(template.FuncMap{}).
		ParseFiles(templatesDir + templateName)
	if err != nil {
		slogLogger.Error("Failed to parse template", "error", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(
				w,
				fmt.Sprintf("Failed to parse template '%s'", templateName),
				http.StatusInternalServerError,
			)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		templateData := TemplateData{
			Title:    "servant",
			Database: convertDatabase(db),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		err = tmpl.ExecuteTemplate(w, templateName, templateData)
		if err != nil {
			slogLogger.Error("Failed to execute template", "error", err)
			http.Error(
				w,
				"Internal Server Error",
				http.StatusInternalServerError,
			)
		}
	}
}
