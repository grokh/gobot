package main

import (
//	"database/sql"
	"html/template"
//	"log"
	"net/http"
	"strings"
)

type Zone struct {
	ZoneAbbr string
	ZoneDisp string
}

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
	Zones   []Zone
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

	query = "SELECT zone_abbr, zone_name FROM zones " + 
		"ORDER BY zone_name ASC"
	stmt, err = db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err := stmt.Query()
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		var z Zone
		err = rows.Scan(&z.ZoneAbbr, &z.ZoneDisp)
		ChkErr(err)
		p.Zones = append(p.Zones, z)
	}

    query = "SELECT attrib_abbr, attrib_display FROM attribs"
	stmt, err = db.Prepare(query)
    ChkErr(err)
    defer stmt.Close()

	rows, err = stmt.Query()
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

func parseForm(p Page, r *http.Request) []string {
	var results []string // slice holding final results of query
	query := "SELECT long_stats FROM items WHERE " // query builder
	var vals []string // slice holding query values

	if r.PostFormValue("itemName") != "" {
		query += "item_name LIKE ? " // TODO this needs a lot of work
		vals = append(vals, "%"+r.PostFormValue("itemName")+"%")
	//	results = FindItem(r.PostFormValue("itemName"), "long_stats") // placeholder for doing it right
	}

	if r.PostFormValue("zoneName") != "" {
		if len(query) > 35 {
			query += " AND "
		}
		query += "from_zone = ? "
		vals = append(vals, r.PostFormValue("zoneName"))
	}

	// TODO add Armor AC somewhere

	if r.PostFormValue("attrib1") != "" {
		if len(query) > 35 {
			query += " AND "
		}
		query += "item_id IN " +
			"(SELECT item_id FROM item_attribs " +
			"WHERE attrib_abbr = ? "
		vals = append(vals, r.PostFormValue("attrib1"))

		if r.PostFormValue("compareAttrib1") != "" {
			if r.PostFormValue("valueAttrib1") != "" {
				switch r.PostFormValue("compareAttrib1") {
				case "gt":
					query += "AND attrib_value > ?"
				case "lt":
					query += "AND attrib_value < ?"
				case "et":
					query += "AND attrib_value = ?"
				}
				vals = append(vals, r.PostFormValue("valueAttrib1"))
			}
		}
		query += ") "
	}

	if r.PostFormValue("attrib2") != "" {
		if len(query) > 35 {
			query += " AND "
		}
		query += "item_id IN " +
			"(SELECT item_id FROM item_attribs " +
			"WHERE attrib_abbr = ?"
		vals = append(vals, r.PostFormValue("attrib2"))

		if r.PostFormValue("compareAttrib2") != "" {
			if r.PostFormValue("valueAttrib2") != "" {
				switch r.PostFormValue("compareAttrib2") {
				case "gt":
					query += "AND attrib_value > ?"
				case "lt":
					query += "AND attrib_value < ?"
				case "et":
					query += "AND attrib_value = ?"
				}
				vals = append(vals, r.PostFormValue("valueAttrib2"))
			}
		}

		query += ") "
	}
	if r.PostFormValue("worn") != "" {
		if len(query) > 35 {
			query += " AND "
		}
		query += "item_id IN " +
			"(SELECT item_id FROM item_slots " +
			"WHERE slot_abbr = ?) "
		vals = append(vals, r.PostFormValue("worn"))
	}

	if r.PostFormValue("type") != "" {
		if len(query) > 35 {
			query += " AND "
		}
		query += "item_id IN " +
			"(SELECT item_id FROM items " +
			"WHERE item_type = ?) "
		vals = append(vals, r.PostFormValue("type"))
	}

	// iterate through restricts
	restricts := []string{
		"!fighter", "!priest", "!thief", "!mage", "!male", "!female",
		"!good", "!neut", "!evil", "!goodrace", "!evilrace",
	}
	for _, v := range restricts {
		if r.PostFormValue(v) != "" {
			if len(query) > 35 {
				query += " AND "
			}
			query += "item_id NOT IN " +
				"(SELECT item_id FROM item_restricts " +
				"WHERE restrict_abbr = ?) "
			vals = append(vals, v)
		}
	}

	// iterate through effects
	for _, v := range p.Effects {
		if r.PostFormValue(v.EffAbbr) != "" {
			if len(query) > 35 {
				query += " AND "
			}
			query += "item_id IN " +
				"(SELECT item_id FROM item_effects " +
				"WHERE effect_abbr = ?) "
			vals = append(vals, v.EffAbbr)
		}
	}

	// iterate through resists
	for _, v := range p.Resists {
		if r.PostFormValue(v.ResAbbr) != "" {
			if len(query) > 35 {
				query += " AND "
			}
			query += "item_id IN " +
				"(SELECT item_id FROM item_resists " +
				"WHERE resist_abbr = ?) "
			vals = append(vals, v.ResAbbr)
		}
	}

	// iterate through flags
	for _, v := range p.Flags {
		if r.PostFormValue(v.FlagAbbr) != "" {
			if len(query) > 35 {
				query += " AND "
			}
			query += "item_id IN " +
				"(SELECT item_id FROM item_flags " +
				"WHERE flag_abbr = ?) "
			vals = append(vals, v.FlagAbbr)
		}
	}

	// iterate through supps
	for _, v := range p.Supps {
		if r.PostFormValue(v.SuppAbbr) != "" {
			if len(query) > 35 {
				query += " AND "
			}
			query += "item_id IN " +
				"(SELECT item_id FROM item_supps " +
				"WHERE supp_abbr = ?) "
			vals = append(vals, v.SuppAbbr)
		}
	}

	if len(query) > 35 {
		switch r.PostFormValue("orderBy") {
		case "name":
			query += "ORDER BY item_name "
		case "zone":
			query += "ORDER BY from_zone "
		case "attrib1": // TODO fix attribs and slot sorting, if possible
			query += "ORDER BY attrib_value "
		case "attrib2":
			query += "ORDER BY attrib_value "
		case "worn":
			query += "ORDER BY slot_abbr "
		case "type":
			query += "ORDER BY item_type "
		}
		switch r.PostFormValue("ascDesc") {
		case "up":
			query += "ASC"
		case "down":
			query += "DESC"
		}

		db := OpenDB()
		defer db.Close()

		stmt, err := db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		args := make([]interface{}, len(vals))
		for n, val := range vals {
			args[n] = val
		}

		rows, err := stmt.Query(args...)
		ChkErr(err)
		defer rows.Close()

		for rows.Next() {
			var s string
			err = rows.Scan(&s)
			ChkErr(err)
			results = append(results, s)
		}
	}
	return results
}

func parseList(r *http.Request) []string {
	var results []string
	txt := r.PostFormValue("list")
	list := strings.Split(strings.Replace(txt, "\r\n", "\n", -1), "\n")

	for _, v := range list {
		if v != "" && !strings.Contains(v, ":") && !strings.Contains(v, "/") {
			if n := strings.Index(v, " ("); n > 0 { // remove everything after the first " ("
				v = v[0:n]
			}
			if n := strings.Index(v, "] "); n > 0 { // remove everything before the first "] "
				v = v[n+1:len(v)]
			}
			if n := strings.Index(v, "   "); n > 0 { // remove everything before the first "   "
				v = v[n+1:len(v)]
			}
			if n := strings.Index(v, "> "); n > 0 { // remove everything before the first "> "
				v = v[n+1:len(v)]
			}
			v = strings.TrimSpace(v) // remove leading and trailing whitespace

			for _, result := range FindItem(v, "long_stats") {
				results = append(results, result)
			}
		}
	}

	return results
}

var tmpl = template.Must(template.ParseFiles(
    "html/index.html",
))

func eqHandler(w http.ResponseWriter, r *http.Request) {
	p := fillStructs()
	if r.Method == "POST" {
		if r.PostFormValue("list") != "" {
			p.Results = parseList(r)
		} else {
			p.Results = parseForm(p, r)
			// TODO make item names javascript link
			// TODO make first / single result full_stats
		}
	}
	err := tmpl.Execute(w, p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

