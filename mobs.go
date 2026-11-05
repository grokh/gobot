package main

import (
	"database/sql"
	"log"
	"strconv"
	"strings"
)

// Kegor's TorilDB.db also has a mobs table: one row per mob, keyed by name,
// with its zone, the eq it's known to carry and the items its quest uses,
// each a list split by |, and his own short_stats and long_stats. He numbers
// mobs that share a name: the bartender 1 to the bartender 20, in as many
// zones.

// mobsDown is the reply to mstat when TorilDB.db, or the mobs table in it,
// is missing or can't be read.
const mobsDown = "Mob stats are unavailable right now. Please try again later."

// mobColumns are the mobs columns gobot reads, which CheckItemDB makes sure
// a new TorilDB.db still has.
var mobColumns = []string{"name", "zone", "equipment", "quests", "short_stats"}

// mobBase is the SQL for a mob's name without the number Kegor gives it:
// the bartender 12 is the bartender.
const mobBase = "rtrim(rtrim(name, '0123456789'), ' ')"

// mobLimit is the most mobs mstat lists.
const mobLimit = 10

// loadsOnLimit is the most mobs astat's Loads on line names. A barbarian
// sword loads on 58.
const loadsOnLimit = 5

// FindMob returns mstat's reply for the mob named in oper: Kegor's short
// stats, with the eq it carries and the items its quest uses.
func FindMob(oper string) []string {
	db, err := OpenItemDB()
	if err != nil {
		log.Printf("%s: %v\n", itemDBFile, err)
		return []string{mobsDown}
	}
	defer db.Close()

	res, err := findMob(db, oper)
	if err != nil {
		// no mobs table, a column gone: say so, rather than that the mob
		// doesn't exist
		log.Printf("%s: mstat %s: %v\n", itemDBFile, oper, err)
		return []string{mobsDown}
	}
	if len(res) == 0 {
		return []string{NotFound("mob", oper)}
	}
	return res
}

// findMob tries each way of matching a mob's name in turn: the whole name,
// number and all; the name without its number, which brings back every mob
// that shares it; then the looser matches stat uses, where the closest name
// comes first, followed by the mobs that share it.
func findMob(db *sql.DB, oper string) ([]string, error) {
	res, err := mobRows(db, "name = ? COLLATE NOCASE", "", oper)
	if err != nil || len(res) > 0 {
		return res, err
	}
	res, err = mobRows(db, mobBase+" = ? COLLATE NOCASE", "", oper)
	if err != nil || len(res) > 0 {
		return res, err
	}
	for _, m := range nameMatches(oper) {
		if m.op != "LIKE" || len(m.args) == 0 {
			continue
		}
		query := "SELECT name, " + mobBase +
			" FROM mobs WHERE short_stats != ''"
		args := make([]interface{}, len(m.args))
		for n, arg := range m.args {
			query += " AND name LIKE ?"
			args[n] = arg
		}
		var name, base string
		err := db.QueryRow(query+" ORDER BY length(name), name LIMIT 1",
			args...).Scan(&name, &base)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		return mobRows(db, mobBase+" = ?", name, base)
	}
	return nil, nil
}

// mobRows returns mstat's lines for up to mobLimit mobs matching test: the
// one named first, if it's there, then the rest, shortest name first, so
// the bartender 2 comes before the bartender 10. If more mobs match, a last
// line says how many.
func mobRows(db *sql.DB, test string, first string, args ...interface{}) (
	[]string, error) {
	rows, err := db.Query("SELECT short_stats, coalesce(equipment, ''), "+
		"coalesce(quests, '') FROM mobs WHERE "+test+" AND short_stats != '' "+
		"ORDER BY name = ? DESC, length(name), name", append(args, first)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []string
	count := 0
	for rows.Next() {
		if count++; count > mobLimit {
			continue
		}
		var stats, eq, quests string
		if err := rows.Scan(&stats, &eq, &quests); err != nil {
			return nil, err
		}
		if items := mobItems(eq); items != "" {
			stats += " * Eq: " + items
		}
		if items := mobItems(quests); items != "" {
			stats += " * Quest: " + items
		}
		res = append(res, tintinSafe(stats))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if more := count - mobLimit; more > 0 {
		res = append(res, "...and "+moreMobs(more)+" by that name.")
	}
	return res, nil
}

// moreMobs is how a reply counts the mobs it leaves out: 1 more mob, 2 more
// mobs.
func moreMobs(n int) string {
	if n == 1 {
		return "1 more mob"
	}
	return strconv.Itoa(n) + " more mobs"
}

// mobItems makes one of Kegor's item lists, split by |, readable.
func mobItems(list string) string {
	var items []string
	for _, item := range strings.Split(list, "|") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return strings.Join(items, ", ")
}

// loadsOn is astat's line for the item called name: the mobs Kegor lists as
// carrying it, with their zones. It's "" if he lists none, or if the mobs
// table can't be read, which costs astat the line but not the item's stats.
// His eq lists don't always match the item's name in case.
func loadsOn(db *sql.DB, name string) string {
	rows, err := db.Query("SELECT name, coalesce(zone, '') FROM mobs "+
		"WHERE instr('|' || lower(coalesce(equipment, '')) || '|', "+
		"'|' || lower(?) || '|') > 0 "+
		"ORDER BY "+mobBase+", length(name), name", name)
	if err != nil {
		log.Printf("%s: loads on %s: %v\n", itemDBFile, name, err)
		return ""
	}
	defer rows.Close()

	var mobs []string
	count := 0
	for rows.Next() {
		var mob, zone string
		if err := rows.Scan(&mob, &zone); err != nil {
			log.Printf("%s: loads on %s: %v\n", itemDBFile, name, err)
			return ""
		}
		if count++; count > loadsOnLimit {
			continue
		}
		if zone != "" {
			mob += " (" + zone + ")"
		}
		mobs = append(mobs, mob)
	}
	if err := rows.Err(); err != nil {
		log.Printf("%s: loads on %s: %v\n", itemDBFile, name, err)
		return ""
	}
	if count == 0 {
		return ""
	}
	line := "Loads on: " + strings.Join(mobs, ", ")
	if more := count - loadsOnLimit; more > 0 {
		line += ", and " + moreMobs(more)
	}
	return tintinSafe(line)
}
