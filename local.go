package main

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// glistBefore is when GlistStats starts counting an identify as recent.
const glistBefore = "2020-11-01"

// GlistStats takes a |-separated glist and lists the items TorilDB.db is
// missing, then those it last had identified before glistBefore, oldest
// first: the ones worth identifying again.
func GlistStats(list string) []string {
	list = strings.Trim(list, "| ")
	list = strings.Replace(list, "|", "\n", -1)

	db, err := OpenItemDB()
	ChkErr(err)
	defer db.Close()

	var txt []string
	for _, name := range listNames(list) {
		found, date := "", ""
		// the name exactly, or the first of its variants, as the site finds it
		for _, n := range []string{name, name + " 1"} {
			var d sql.NullString
			err := db.QueryRow("SELECT name, date FROM eq WHERE name = ?", n).
				Scan(&found, &d)
			if err == sql.ErrNoRows {
				continue
			}
			ChkErr(err)
			date = d.String
			break
		}
		switch {
		case found == "":
			txt = append(txt, NotFound("item", name)+"\n")
		case date == "":
			txt = append(txt, fmt.Sprintf("Last ID: unknown - %s\n", found))
		case date < glistBefore:
			txt = append(txt, fmt.Sprintf("Last ID: %s - %s\n", date, found))
		}
	}
	sort.Strings(txt)
	return txt
}
