package main

import (
	"embed"
	"html/template"
)

//go:embed web/*
var webFS embed.FS

var (
	tplIndex  *template.Template
	tplView   *template.Template
	tplSearch *template.Template
	tplScan   *template.Template
)

func mustLoadWebTemplates() {
	tplIndex = template.Must(template.ParseFS(webFS, "web/index.html"))
	tplView = template.Must(template.ParseFS(webFS, "web/view.html"))
	tplSearch = template.Must(template.ParseFS(webFS, "web/partials_search.html"))
	tplScan = template.Must(template.ParseFS(webFS, "web/scan.html"))
}

