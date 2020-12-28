package main

import (
	"fmt"
	"html/template"
	"net/http"
)

type Page struct {
	Title   string
	Date    string
	Results []string
}

var templates = template.Must(template.ParseFiles(
	"html/index.html",
))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func eqHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Title: "title", Date: "2020-12-27"}
	switch r.URL.Path[9:] {
	case "", "index.php", "advanced.php", "list.php", "index.html":
		p.Title = "TorilMUD Equipment Database"
		if r.Method == "POST" {
			results := FindItem(r.PostFormValue("itemName"), "short_stats")
			p.Results = results
		}
		renderTemplate(w, "index.html", p)
	}
}
