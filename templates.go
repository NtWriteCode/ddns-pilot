package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

var templates *template.Template

func initTemplates() {
	var err error
	templates, err = template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func setupStaticHandler() {
	// Serve static files (CSS, JS, images)
	http.Handle("/static/", http.FileServer(http.FS(staticFS)))
}
