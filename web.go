package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
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
	Link    string // to the search that found Results
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

// formValue is the search field called key: from the form for a POST, or
// from the address for a GET, never a mix. A box left unticked isn't sent,
// so a POST that read the address too could pick up one ticked in the link
// it was sent from.
func formValue(r *http.Request, key string) string {
	if r.Method == "POST" {
		return r.PostFormValue(key)
	}
	return r.FormValue(key)
}

// formKeys are the names of the TorilEQ form's search fields: the ones
// parseForm reads, and searchLink puts in a link.
func formKeys() []string {
	keys := []string{"itemName", "zoneName", "attrib1", "compareAttrib1",
		"valueAttrib1", "attrib2", "compareAttrib2", "valueAttrib2", "worn",
		"type", "orderBy", "ascDesc", "format"}
	for _, u := range usableBy {
		keys = append(keys, u.abbr)
	}
	for _, e := range effects {
		keys = append(keys, e.abbr)
	}
	for _, res := range resists {
		keys = append(keys, res.abbr)
	}
	for _, f := range flags {
		keys = append(keys, f.abbr)
	}
	for _, s := range supps {
		keys = append(keys, s.abbr)
	}
	return keys
}

// parseForm searches TorilDB.db for the items matching every field filled
// in on the TorilEQ form, sent by POST or, from a link, by GET.
func parseForm(r *http.Request) []string {
	length := "long_stats" // the form's default; also covers a missing value
	if formValue(r, "format") == "short" {
		length = "short_stats"
	}

	var tests []string
	var args []interface{}
	add := func(test string, targs []interface{}) {
		tests = append(tests, test)
		args = append(args, targs...)
	}

	if name := formValue(r, "itemName"); name != "" {
		add("name LIKE ?", vals("%"+name+"%"))
	}

	if zone := formValue(r, "zoneName"); zone != "" {
		// the zone, or one of the zones of an item that loads in several
		add("instr(? || zone || ?, ?) > 0",
			vals(zoneSep, zoneSep, zoneSep+zone+zoneSep))
	}

	for _, n := range []string{"1", "2"} {
		abbr := formValue(r, "attrib"+n)
		if abbr == "" {
			continue
		}
		s, ok := findStat(abbr)
		if !ok {
			add(noMatch, nil)
			continue
		}
		cmp, ok := compareSQL[formValue(r, "compareAttrib"+n)]
		if v := formValue(r, "valueAttrib"+n); ok && v != "" {
			add(col(s.col)+cmp, vals(v))
		} else {
			add(col(s.col)+" != 0", nil) // has the attribute at all
		}
	}

	if worn := formValue(r, "worn"); worn != "" {
		if s, ok := findSlot(worn); ok {
			add(hasToken("worn", s.worn...))
		} else {
			add(noMatch, nil)
		}
	}

	if t := formValue(r, "type"); t != "" {
		add("type = ?", vals(t))
	}

	// Usable By: leave out items with the restriction
	for _, u := range usableBy {
		if formValue(r, u.abbr) != "" {
			test, targs := hasToken("item_flags", u.restrict)
			add("NOT "+test, targs)
		}
	}

	for _, e := range effects {
		if formValue(r, e.abbr) != "" {
			add(hasToken("affect_flags", e.token))
		}
	}

	for _, res := range resists {
		if formValue(r, res.abbr) != "" {
			add(col(res.col)+" > 0", nil)
		}
	}

	for _, f := range flags {
		if formValue(r, f.abbr) != "" {
			add(f.match())
		}
	}

	for _, s := range supps {
		if formValue(r, s.abbr) != "" {
			add(s.test, nil)
		}
	}

	if len(tests) == 0 {
		return nil
	}
	query := "SELECT " + length + " FROM eq WHERE " +
		strings.Join(tests, " AND ")
	if order, oargs := formOrder(r); order != "" {
		query += " ORDER BY " + order
		args = append(args, oargs...)
	}

	db, err := OpenItemDB()
	ChkErr(err)
	defer db.Close()
	results, _, err := queryStats(db, query, 0, args...)
	ChkErr(err)
	return results
}

// wornOrder is the ORDER BY term for the form's Worn Slot choice, with its ?
// values: an item's place in slots, so results run in the order of the
// form's Worn Slot list. An item worn on several slots goes with the first
// of them in that list, and one with no slot gobot knows goes last.
func wornOrder() (string, []interface{}) {
	order := "CASE"
	var args []interface{}
	for n, s := range slots {
		test, targs := hasToken("worn", s.worn...)
		order += fmt.Sprintf(" WHEN %s THEN %d", test, n)
		args = append(args, targs...)
	}
	return order + fmt.Sprintf(" ELSE %d END", len(slots)), args
}

// formOrder is the ORDER BY for the form's Order By choice, then name, with
// its ? values: a column, or the stat chosen as Attribute 1 or 2 (name if
// that was left empty), or the worn slot. It's "" for a choice the form
// doesn't offer.
func formOrder(r *http.Request) (string, []interface{}) {
	by := formValue(r, "orderBy")
	order, ok := orderSQL[by]
	var args []interface{}
	switch by {
	case "attrib1", "attrib2":
		order, ok = "name", true
		if s, found := findStat(formValue(r, by)); found {
			order = col(s.col)
		}
	case "worn":
		order, args = wornOrder()
		ok = true
	}
	if !ok {
		return "", nil
	}
	switch formValue(r, "ascDesc") {
	case "up":
		order += " ASC"
	case "down":
		order += " DESC"
	}
	if !strings.HasPrefix(order, "name") {
		order += ", name"
	}
	return order, args
}

// searchLink is a link to the search r asked for, which the site runs by
// GET: the fields named in keys that were filled in. It's "" if none were.
func searchLink(r *http.Request, keys []string) string {
	v := url.Values{}
	for _, k := range keys {
		if val := formValue(r, k); val != "" {
			v.Set(k, val)
		}
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

// webFstat runs the fstat search in oper for the site, as the links fstat
// replies with ask it to: every item found, in fstat's order, in short
// stats unless format is long.
func webFstat(oper string, format string) []string {
	length := "short_stats"
	if format == "long" {
		length = "long_stats"
	}
	where, args, order, ok := fstatSearch(oper)
	if !ok || where == "" {
		return []string{NotFound("item(s)", oper)}
	}

	db, err := OpenItemDB()
	ChkErr(err)
	defer db.Close()
	results, _, err := queryStats(db, "SELECT "+length+" FROM eq WHERE "+
		where+" ORDER BY "+order, 0, args...)
	ChkErr(err)
	if len(results) == 0 {
		return []string{NotFound("item(s)", oper)}
	}
	return results
}

// exactItem returns long stats for the item with exactly the name given,
// or, failing that, for the first of its variants, as in "a sunrod 1" (see
// eqName).
func exactItem(db *sql.DB, name string) string {
	found, err := eqName(db, name)
	ChkErr(err)
	if found == "" {
		return NotFound("item", name)
	}
	var text sql.NullString
	ChkErr(db.QueryRow("SELECT long_stats FROM eq WHERE name = ?", found).
		Scan(&text))
	return text.String
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
	switch {
	case r.Method == "POST" && r.PostFormValue("list") != "":
		p.Results = ParseList(r.PostFormValue("list"))
	case formValue(r, "fstat") != "":
		// the link an fstat reply ends with: the form has no fstat field
		p.Results = webFstat(formValue(r, "fstat"), formValue(r, "format"))
		p.Link = searchLink(r, []string{"fstat", "format"})
	case r.Method == "POST" || r.URL.RawQuery != "":
		p.Results = parseForm(r)
		if p.Results != nil {
			p.Link = searchLink(r, formKeys())
		}
		// TODO make item names javascript link
		// TODO make first / single result full_stats
	}
	err := tmpl.Execute(w, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
