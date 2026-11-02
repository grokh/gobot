package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
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

// column reads a one-column query into a slice.
func column(db *sql.DB, query string) []string {
	rows, err := db.Query(query)
	ChkErr(err)
	var vals []string
	for rows.Next() {
		var v string
		ChkErr(rows.Scan(&v))
		vals = append(vals, v)
	}
	ChkRows(rows)
	return vals
}

// fillStructs gathers what the TorilEQ form offers: zones and item types
// from TorilDB.db, everything else from the tables in items.go.
func fillStructs() Page {
	var p Page

	db, err := OpenItemDB()
	ChkErr(err)
	defer db.Close()

	var date sql.NullString
	ChkErr(db.QueryRow("SELECT MAX(date) FROM eq").Scan(&date))
	p.Date = date.String

	for _, z := range zoneNames(column(db,
		"SELECT DISTINCT zone FROM eq WHERE zone != ''")) {
		p.Zones = append(p.Zones, Zone{z, z})
	}
	for _, t := range column(db,
		"SELECT DISTINCT type FROM eq WHERE type != ''") {
		p.Types = append(p.Types, IType{t, typeName(t)})
	}
	sort.Slice(p.Types, func(a, b int) bool {
		return p.Types[a].TypeDisp < p.Types[b].TypeDisp
	})

	for _, s := range stats {
		p.Attribs = append(p.Attribs, Attrib{s.abbr, s.display})
	}
	for _, s := range slots {
		p.Slots = append(p.Slots, Slot{s.abbr, s.display})
	}
	for _, e := range effects {
		p.Effects = append(p.Effects, Effect{e.abbr, e.display})
	}
	for _, r := range resists {
		p.Resists = append(p.Resists, Resist{r.abbr, r.display})
	}
	for _, f := range flags {
		p.Flags = append(p.Flags, Flag{f.abbr, f.display})
	}
	for _, s := range supps {
		p.Supps = append(p.Supps, Supp{s.abbr, s.display})
	}
	return p
}

// compareSQL maps the form's attribute comparisons to SQL. Any other value
// is ignored, so the query never gets a ? without a value to go with it.
var compareSQL = map[string]string{
	"gt": " > ?",
	"lt": " < ?",
	"et": " = ?",
}

// orderSQL maps the form's Order By choices to eq columns.
var orderSQL = map[string]string{
	"name": "name",
	"zone": "zone",
	"type": "type",
}

// vals makes a ? argument list.
func vals(v ...interface{}) []interface{} {
	return v
}

// noMatch is the test for a value the form never offers, like an attribute
// it doesn't list: it matches nothing, rather than being left out.
const noMatch = "0"

// parseForm searches TorilDB.db for the items matching every field filled
// in on the TorilEQ form.
func parseForm(r *http.Request) []string {
	length := "long_stats" // the form's default; also covers a missing value
	if r.PostFormValue("format") == "short" {
		length = "short_stats"
	}

	var tests []string
	var args []interface{}
	add := func(test string, targs []interface{}) {
		tests = append(tests, test)
		args = append(args, targs...)
	}

	if name := r.PostFormValue("itemName"); name != "" {
		add("name LIKE ?", vals("%"+name+"%")) // TODO needs a lot of work
	}

	if zone := r.PostFormValue("zoneName"); zone != "" {
		// the zone, or one of the zones of an item that loads in several
		add("instr(? || zone || ?, ?) > 0",
			vals(zoneSep, zoneSep, zoneSep+zone+zoneSep))
	}

	for _, n := range []string{"1", "2"} {
		abbr := r.PostFormValue("attrib" + n)
		if abbr == "" {
			continue
		}
		s, ok := findStat(abbr)
		if !ok {
			add(noMatch, nil)
			continue
		}
		cmp, ok := compareSQL[r.PostFormValue("compareAttrib"+n)]
		if v := r.PostFormValue("valueAttrib" + n); ok && v != "" {
			add(col(s.col)+cmp, vals(v))
		} else {
			add(col(s.col)+" != 0", nil) // has the attribute at all
		}
	}

	if worn := r.PostFormValue("worn"); worn != "" {
		if s, ok := findSlot(worn); ok {
			add(hasToken("worn", s.worn...))
		} else {
			add(noMatch, nil)
		}
	}

	if t := r.PostFormValue("type"); t != "" {
		add("type = ?", vals(t))
	}

	// Usable By: leave out items with the restriction
	for _, u := range usableBy {
		if r.PostFormValue(u.abbr) != "" {
			test, targs := hasToken("item_flags", u.restrict)
			add("NOT "+test, targs)
		}
	}

	for _, e := range effects {
		if r.PostFormValue(e.abbr) != "" {
			add(hasToken("affect_flags", e.token))
		}
	}

	for _, res := range resists {
		if r.PostFormValue(res.abbr) != "" {
			add(col(res.col)+" > 0", nil)
		}
	}

	for _, f := range flags {
		if r.PostFormValue(f.abbr) != "" {
			add(f.match())
		}
	}

	for _, s := range supps {
		if r.PostFormValue(s.abbr) != "" {
			add(s.test, nil)
		}
	}

	if len(tests) == 0 {
		return nil
	}
	query := "SELECT " + length + " FROM eq WHERE " +
		strings.Join(tests, " AND ")
	// TODO attrib1, attrib2 and worn sorting
	if order, ok := orderSQL[r.PostFormValue("orderBy")]; ok {
		query += " ORDER BY " + order
		switch r.PostFormValue("ascDesc") {
		case "up":
			query += " ASC"
		case "down":
			query += " DESC"
		}
		if order != "name" {
			query += ", name"
		}
	}

	db, err := OpenItemDB()
	ChkErr(err)
	defer db.Close()

	rows, err := db.Query(query, args...)
	ChkErr(err)
	var results []string
	for rows.Next() {
		var s sql.NullString
		ChkErr(rows.Scan(&s))
		results = append(results, s.String)
	}
	ChkRows(rows)
	return results
}

// exactItem returns long stats for the item with exactly the name given,
// or, failing that, for the first of its variants, as in "a sunrod 1".
func exactItem(db *sql.DB, name string) string {
	for _, n := range []string{name, name + " 1"} {
		var text sql.NullString
		err := db.QueryRow("SELECT long_stats FROM eq WHERE name = ?", n).
			Scan(&text)
		if err == nil {
			return text.String
		}
		if err != sql.ErrNoRows {
			fatal(err)
		}
	}
	return NotFound("item", name)
}

// listNames picks the item names out of output from 'glist', 'equip',
// 'auction list', 'inventory', 'look in bag', or 'storage list'.
func listNames(txt string) []string {
	var names []string
	list := strings.Split(strings.Replace(txt, "\r\n", "\n", -1), "\n")

	for _, v := range list {
		if v != "" && !strings.Contains(v, ":") && !strings.Contains(v, "h/") &&
			!strings.Contains(v, ";") && !strings.Contains(v, "[RETURN") &&
			!strings.Contains(v, "======") && !strings.Contains(v, "#") {
			// to parse auction results
			if n := strings.Index(v, ")     "); n > 0 {
				x := strings.Count(v, "'")
				if n+12 <= 48+x && 48+x <= len(v) { // short lines: leave as is
					v = v[n+12 : 48+x]
				}
			}
			// remove everything after the first " ("
			// to remove (illuminating) or whatever from the end of items
			if n := strings.Index(v, " ("); n > 0 {
				v = v[0:n]
			}
			// remove everything before the first "] "
			// to parse combined [5]-style items from inv or look in
			if n := strings.Index(v, "] "); n > 0 {
				v = v[n+1 : len(v)]
			}
			// remove everything before the first "   "
			// to parse items from a 'glist'
			if n := strings.Index(v, "   "); n > 0 {
				v = v[n+1 : len(v)]
			}
			// remove everything before the first "> "
			// to parse items from an 'eq' list
			if n := strings.Index(v, "> "); n > 0 {
				v = v[n+1 : len(v)]
			}
			// remove leading and trailing whitespace
			if v = strings.TrimSpace(v); v != "" {
				names = append(names, v)
			}
		}
	}
	return names
}

// ParseList returns long stats for each item in a pasted list.
func ParseList(txt string) []string {
	db, err := OpenItemDB()
	ChkErr(err)
	defer db.Close()

	var results []string
	for _, name := range listNames(txt) {
		results = append(results, exactItem(db, name))
	}
	return results
}

// tmpl is the TorilEQ page. Only the web server (-port) uses it, so main
// loads it there with loadPage, and the bot's commands don't need html/.
var tmpl *template.Template

func loadPage() {
	tmpl = template.Must(template.ParseFiles("html/index.html"))
}

func eqHandler(w http.ResponseWriter, r *http.Request) {
	// Errors while serving a request panic (see fatal); answer them with a
	// 500 instead of letting one bad request take the whole site down.
	defer func() {
		if err := recover(); err != nil {
			log.Printf("web: %s %s failed: %v", r.Method, r.URL.Path, err)
			http.Error(w, "Sorry, that search failed. Please try again.",
				http.StatusInternalServerError)
		}
	}()
	if _, err := os.Stat(itemDBFile); err != nil {
		log.Printf("web: %s: %v", itemDBFile, err)
		http.Error(w, itemsDown, http.StatusServiceUnavailable)
		return
	}
	p := fillStructs()
	if r.Method == "POST" {
		if r.PostFormValue("list") != "" {
			p.Results = ParseList(r.PostFormValue("list"))
		} else {
			p.Results = parseForm(r)
			// TODO make item names javascript link
			// TODO make first / single result full_stats
		}
	}
	err := tmpl.Execute(w, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
