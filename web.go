package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
)

type Page struct {
	Date    string
	Results []string
}

var templates = template.Must(template.ParseFiles(
	"html/index.html",
))

func mostRecent() string {
	db := OpenDB()
    defer db.Close()

	query := "SELECT MAX(last_id) FROM items"
	stmt, err := db.Prepare(query)
    ChkErr(err)
    defer stmt.Close()

	var txt string
	err = stmt.QueryRow().Scan(&txt)
	if err == sql.ErrNoRows {
		txt = "Database Lost"
	} else if err != nil {
		log.Fatal(err)
	}
	return txt
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func eqHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Date: mostRecent()}
	if r.Method == "POST" {
		results := FindItem(r.PostFormValue("itemName"), "short_stats")
		p.Results = results
	}
	renderTemplate(w, "index.html", p)
}
