package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/opd-ai/dndbot/srv/util"
)

func HandleGenerate(w http.ResponseWriter, r *http.Request) {

}

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(TemplateFS, "templates/index.html")
	if err != nil {
		util.ErrorLogger.Printf("Template parsing error: %v", err)
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		util.ErrorLogger.Printf("Template execution error: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func HandleDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	filePath := filepath.Join("output", sessionID, "adventure.zip")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		util.ErrorLogger.Printf("Download file not found: %s", filePath)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=adventure-%s.zip", sessionID))
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, filePath)
	util.InfoLogger.Printf("File downloaded: %s", filePath)
}
