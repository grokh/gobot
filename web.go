package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
)

type Page struct {
	Date string
	Res  []string
}

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

var tmpl = template.Must(template.ParseFiles(
    "html/index.html",
))

func eqHandler(w http.ResponseWriter, r *http.Request) {
	p := Page{
		Date: mostRecent(),
		Res: nil,
	}
	if r.Method == "POST" {
		p.Res = FindItem(r.PostFormValue("itemName"), "short_stats")
	}
	err := tmpl.Execute(w, p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
