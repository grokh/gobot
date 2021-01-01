package main

import (
//	"database/sql"
	"html/template"
//	"log"
	"net/http"
)

type Attrib struct {
	AttAbbr string
	AttDisp string
}

type Slot struct {
	SlotAbbr string
	SlotDisp string
}

type IType struct {
	TypeAbbr string
	TypeDisp string
}

type Effect struct {
	EffAbbr string
	EffDisp string
}

type Resist struct {
	ResAbbr string
	ResDisp string
}

type Flag struct {
	FlagAbbr string
	FlagDisp string
}

type Supp struct {
	SuppAbbr string
	SuppDisp string
}

type Page struct {
	Date    string
	Results []string
	Attribs []Attrib
	Slots   []Slot
	Types   []IType
	Effects []Effect
	Resists []Resist
	Flags   []Flag
	Supps   []Supp
}

func fillStructs() Page {
	p := Page{
        Date: "2020-12-29",
    }

	db := OpenDB()
    defer db.Close()

	query := "SELECT MAX(last_id) FROM items"
    stmt, err := db.Prepare(query)
    ChkErr(err)
    defer stmt.Close()

    err = stmt.QueryRow().Scan(&p.Date)
	ChkErr(err)

    query = "SELECT attrib_abbr, attrib_display FROM attribs"
	stmt, err = db.Prepare(query)
    ChkErr(err)
    defer stmt.Close()

	rows, err := stmt.Query()
    ChkErr(err)
    defer rows.Close()

	for rows.Next() {
		var a Attrib
		err = rows.Scan(&a.AttAbbr, &a.AttDisp)
		ChkErr(err)
		p.Attribs = append(p.Attribs, a)
	}

	query = "SELECT slot_abbr, slot_display FROM slots"
	stmt, err = db.Prepare(query)
    ChkErr(err)
    defer stmt.Close()

    rows, err = stmt.Query()
    ChkErr(err)
    defer rows.Close()

    for rows.Next() {
		var s Slot
		err = rows.Scan(&s.SlotAbbr, &s.SlotDisp)
		ChkErr(err)
		p.Slots = append(p.Slots, s)
	}

	query = "SELECT type_abbr, type_display FROM item_types"
	stmt, err = db.Prepare(query)
    ChkErr(err)
    defer stmt.Close()

    rows, err = stmt.Query()
    ChkErr(err)
    defer rows.Close()

    for rows.Next() {
		var t IType
		err = rows.Scan(&t.TypeAbbr, &t.TypeDisp)
		ChkErr(err)
		p.Types = append(p.Types, t)
	}

	query = "SELECT effect_abbr, effect_display FROM effects"
    stmt, err = db.Prepare(query)
    ChkErr(err)
    defer stmt.Close()

    rows, err = stmt.Query()
    ChkErr(err)
    defer rows.Close()

    for rows.Next() {
        var e Effect
        err = rows.Scan(&e.EffAbbr, &e.EffDisp)
        ChkErr(err)
        p.Effects = append(p.Effects, e)
    }

	query = "SELECT resist_abbr, resist_display FROM resists"
    stmt, err = db.Prepare(query)
    ChkErr(err)
    defer stmt.Close()

    rows, err = stmt.Query()
    ChkErr(err)
    defer rows.Close()

    for rows.Next() {
        var r Resist
        err = rows.Scan(&r.ResAbbr, &r.ResDisp)
        ChkErr(err)
        p.Resists = append(p.Resists, r)
    }

	query = "SELECT flag_abbr, flag_display FROM flags"
    stmt, err = db.Prepare(query)
    ChkErr(err)
    defer stmt.Close()

    rows, err = stmt.Query()
    ChkErr(err)
    defer rows.Close()

    for rows.Next() {
        var f Flag
        err = rows.Scan(&f.FlagAbbr, &f.FlagDisp)
        ChkErr(err)
        p.Flags = append(p.Flags, f)
    }

	query = "SELECT supp_abbr, supp_display FROM supps"
    stmt, err = db.Prepare(query)
    ChkErr(err)
    defer stmt.Close()

    rows, err = stmt.Query()
    ChkErr(err)
    defer rows.Close()

    for rows.Next() {
        var s Supp
        err = rows.Scan(&s.SuppAbbr, &s.SuppDisp)
        ChkErr(err)
        p.Supps = append(p.Supps, s)
    }

	return p
}

func fillResults(r *http.Request) []string {
	var res []string
	sql := "SELECT long_stats FROM items WHERE "
	if r.PostFormValue("itemName") != "" {
		sql += "item_name LIKE ?"
		res = FindItem(r.PostFormValue("itemName"), "long_stats")
	}
	return res
}

var tmpl = template.Must(template.ParseFiles(
    "html/index.html",
))

func eqHandler(w http.ResponseWriter, r *http.Request) {
	p := fillStructs()
	if r.Method == "POST" {
		p.Results = fillResults(r)
	}
	err := tmpl.Execute(w, p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

