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

// baseName is the SQL for a mob's or an item's name without the number
// Kegor gives it: the bartender 12 is the bartender, and a longsword 2 is a
// longsword.
const baseName = "rtrim(rtrim(name, '0123456789'), ' ')"

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
	res, err = mobRows(db, baseName+" = ? COLLATE NOCASE", "", oper)
	if err != nil || len(res) > 0 {
		return res, err
	}
	for _, m := range nameMatches(oper) {
		if m.op != "LIKE" || len(m.args) == 0 {
			continue
		}
		query := "SELECT name, " + baseName +
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
		return mobRows(db, baseName+" = ?", name, base)
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
		res = append(res, "...and "+countMore(more, "mob")+" by that name.")
	}
	return res, nil
}

// countMore is how a reply counts the mobs or items it leaves out: 1 more
// mob, 2 more items.
func countMore(n int, thing string) string {
	if n == 1 {
		return "1 more " + thing
	}
	return strconv.Itoa(n) + " more " + thing + "s"
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
func loadsOn(db *sql.DB, name string) string {
	mobs, err := carriers(db, name)
	if err != nil {
		log.Printf("%s: loads on %s: %v\n", itemDBFile, name, err)
		return ""
	}
	if len(mobs) == 0 {
		return ""
	}
	line := "Loads on: " +
		strings.Join(mobs[:min(len(mobs), loadsOnLimit)], ", ")
	if more := len(mobs) - loadsOnLimit; more > 0 {
		line += ", and " + countMore(more, "mob")
	}
	return tintinSafe(line)
}

// namedIn is the SQL test for a mob whose eq list, split by |, names the
// item given. His lists don't always match an item's name in case.
const namedIn = "instr('|' || lower(coalesce(equipment, '')) || '|', " +
	"'|' || lower(?) || '|') > 0"

// carriers are the mobs that carry the item called name, as "mob (zone)":
// first those whose eq lists name it, then, for an item Kegor numbers, those
// whose lists give it the name the game shows, without his number, if their
// zones say which item of that name they carry (itemZones.picks). His lists
// mostly do that: a short sword, not a short sword 2. Each kind comes in
// order of name, the bartender 2 before the bartender 10.
func carriers(db *sql.DB, name string) ([]string, error) {
	query := "SELECT name, coalesce(zone, ''), " + namedIn + " AS exact " +
		"FROM mobs WHERE " + namedIn
	args := []interface{}{name, name}
	var zones itemZones
	base := strings.TrimRight(strings.TrimRight(name, "0123456789"), " ")
	if endsInNumber(name) && base != "" {
		var err error
		if zones, err = namesakeZones(db, name, base); err != nil {
			return nil, err
		}
		if zones.own != "" {
			query += " OR " + namedIn
			args = append(args, base)
		}
	}
	rows, err := db.Query(query+" ORDER BY exact DESC, "+baseName+
		", length(name), name", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mobs []string
	for rows.Next() {
		var mob, zone string
		var exact bool
		if err := rows.Scan(&mob, &zone, &exact); err != nil {
			return nil, err
		}
		if !exact && !zones.picks(zone) {
			continue
		}
		if zone != "" {
			mob += " (" + zone + ")"
		}
		mobs = append(mobs, mob)
	}
	return mobs, rows.Err()
}

// itemZones are the zones an item Kegor numbers loads in, as eq's zone
// column gives them, and those of the other items of its name, numbered or
// not.
type itemZones struct {
	own    string
	others []string
}

// namesakeZones finds the itemZones of the item called name, which is base
// and a number. Its namesakes are the items sameName lists with it.
func namesakeZones(db *sql.DB, name, base string) (itemZones, error) {
	var zones itemZones
	rows, err := db.Query("SELECT name, coalesce(zone, '') FROM eq WHERE "+
		baseName+" = ? COLLATE NOCASE", base)
	if err != nil {
		return zones, err
	}
	defer rows.Close()
	for rows.Next() {
		var item, zone string
		if err := rows.Scan(&item, &zone); err != nil {
			return zones, err
		}
		if item == name {
			zones.own = zone
		} else {
			zones.others = append(zones.others, zone)
		}
	}
	return zones, rows.Err()
}

// picks reports whether a mob in zone whose eq list gives the item's name
// without the number carries this item: its zone shares a zone with this
// item's, and with no other item's of that name. A mob where two of them
// load could carry either, so it counts for neither.
func (z itemZones) picks(zone string) bool {
	if !sharesZone(z.own, zone) {
		return false
	}
	for _, other := range z.others {
		if sharesZone(other, zone) {
			return false
		}
	}
	return true
}

// sharesZone reports whether two zone values name a zone in common, in any
// case. Either can name several, as Silverymoon - Westgate does.
func sharesZone(a, b string) bool {
	for _, x := range strings.Split(a, zoneSep) {
		if x = strings.TrimSpace(x); x == "" {
			continue
		}
		for _, y := range strings.Split(b, zoneSep) {
			if strings.EqualFold(x, strings.TrimSpace(y)) {
				return true
			}
		}
	}
	return false
}
