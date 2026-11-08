package main

import (
	"bytes"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// dbcopy.sh publishes toril.db for NyyLIB, the Mudlet package, with its
// items and item_flags refilled from TorilDB.db by dbcopy_items.sql. These
// tests check them as NyyLIB reads them.

// nyyEq is a test TorilDB.db's eq table, in Kegor's formats, with the cases
// dbcopy_items.sql treats differently.
var nyyEq = []dbRow{
	// labelled by NyyLIB, from the old form of the potion, scroll or poison
	// part
	{"name": "a scarlet potion", "keywords": "scarlet potion",
		"zone": "Ribcage", "type": "POTION", "worn": "NOBITS", "wt": 1,
		"value": 1500, "date": "2024-03-01", "s_level": 41,
		"s_spell": "harm - heal", "item_flags": "MAGIC",
		"short_stats": "a scarlet potion * POTION (Level 41 harm - heal) " +
			"* Wt:1 Val:1.5p * Ribcage"},
	{"name": "a rolled scroll", "type": "SCROLL", "s_level": 50,
		"s_spell": "bless - armor - remove curse",
		"short_stats": "a rolled scroll * SCROLL (Level 50 bless - armor - " +
			"remove curse) * Wt:2 Val:5p * Mithril Hall (S)"},
	{"name": "a vial of tree frog toxin", "type": "POISON",
		"p_poison": "fumble", "p_level": 21, "p_apps": 5, "p_hits": 3,
		"short_stats": "a vial of tree frog toxin * POISON (Level 21 " +
			"fumble, 5 Apps, 3 Hits) * Wt:1 Val:10p * Skullport (S)"},
	{"name": "a vial of black adder venom", "type": "POISON",
		"p_poison": "enervate", "p_level": 40, "p_apps": 1,
		"short_stats": "a vial of black adder venom * POISON (Level 40 " +
			"enervate, 1 Apps) * Wt:1 Val:2p * Skullport (S)"},
	// not labelled, since the old form wouldn't fit: a spell of '?', one
	// with an apostrophe, no level, nothing after the potion part, and no
	// stats at all
	{"name": "a suicide potion", "type": "POTION", "s_level": 1,
		"s_spell": "?", "short_stats": "a suicide potion * !ID * POTION " +
			"(Level 1 ?) * QUEST-ITEM * Comarian Mines (Q)"},
	{"name": "a vial of mystra's tears", "type": "POTION", "s_level": 30,
		"s_spell": "mystra's blessing", "short_stats": "a vial of " +
			"mystra's tears * POTION (Level 30 mystra's blessing) * Wt:1 " +
			"Val:0 * Ako Village"},
	{"name": "a potion of clarity", "type": "POTION",
		"short_stats": "a potion of clarity * !ID * POTION * Hyssk (Q)"},
	{"name": "a cloudy potion", "type": "POTION", "s_level": 10,
		"s_spell": "heal", "short_stats": "a cloudy potion * POTION " +
			"(Level 10 heal)"},
	{"name": "a murky potion", "type": "POTION"},
	// weapon buttons: bows, crossbows and slings are 'ranged', and 2H is
	// two-handed
	{"name": "an ironwood bow", "type": "WEAPON", "w_type": "Longbow",
		"worn": "WIELD 2H", "item_flags": "NO-CLERIC NO-MAGE",
		"short_stats": "an ironwood bow (WIELD 2H) * 4D6 3%/3x * !PRIEST " +
			"!MAGE * Wt:5 Val:0 * Muspelheim (R)"},
	{"name": "a well-made crossbow", "type": "WEAPON",
		"w_type": "Heavy Crossbow", "worn": "WIELD 2H",
		"short_stats": "a well-made crossbow (WIELD 2H) * 4D6 2/2 6%/2x " +
			"* Wt:5 Val:1p * Ribcage"},
	{"name": "a leather sling", "type": "WEAPON", "w_type": "Sling",
		"worn": "WIELD", "short_stats": "a leather sling (WIELD) * 1D4 " +
			"* Wt:1 Val:5s * Hulburg (S)"},
	{"name": "a heavy throwing dart", "type": "WEAPON",
		"w_type": "Throwing Dagger", "worn": "WIELD THROW",
		"short_stats": "a heavy throwing dart (WIELD THROW) * 1D3 3%/3x " +
			"* Wt:7 Val:1.8p * Scardale"},
	{"name": "a black longsword of destruction", "type": "WEAPON",
		"w_type": "Longsword", "worn": "WIELD 2H",
		"item_flags":  "MAGIC FLOAT NOBURN NOLOCATE NO-CLERIC NO-THIEF",
		"short_stats": swordShort},
	{"name": "the infernal stiletto of bane", "type": "WEAPON",
		"w_type": "Dagger", "worn": "WIELD",
		"item_flags":  "MAGIC FLOAT NOBURN NOLOCATE NO-WARRIOR NO-CLERIC",
		"short_stats": stilettoShort},
	// other types and flags
	{"name": "a wooden druid's staff", "type": "STAFF", "worn": "HOLD 2H",
		"item_flags": "NODROP BLESS SECRET NO-WARRIOR",
		"short_stats": "a wooden druid's staff (HOLD 2H) * Wt:4 Val:0 " +
			"* Darktree"},
	{"name": "a suit of lordly plate", "type": "ARMOR",
		"worn": "WHOLE-BODY", "item_flags": "MAGIC NORENT",
		"short_stats": suitShort},
	{"name": "a jug of water", "type": "LIQUID_CONT",
		"short_stats": "a jug of water * Wt:2 Val:1s * Waterdeep (S)"},
	{"name": "a tiny kitten", "type": "PET",
		"short_stats": "a tiny kitten * Wt:0 Val:0 * Waterdeep (S)"},
	// NyyLIB takes the first item whose name has the one it looks up in it,
	// and Kegor has the longer one first here
	{"name": "a tiny mithril stud set with a ruby", "type": "WORN",
		"worn": "EARRING", "short_stats": rubyShort},
	{"name": "a tiny mithril stud", "type": "WORN", "worn": "EARRING",
		"short_stats": studShort},
}

// nyyTypes is the item_type of each item in nyyEq.
var nyyTypes = map[string]string{
	"a scarlet potion":                    "potion",
	"a rolled scroll":                     "scroll",
	"a vial of tree frog toxin":           "poison",
	"a vial of black adder venom":         "poison",
	"a suicide potion":                    "potion",
	"a vial of mystra's tears":            "potion",
	"a potion of clarity":                 "potion",
	"a cloudy potion":                     "potion",
	"a murky potion":                      "potion",
	"an ironwood bow":                     "ranged",
	"a well-made crossbow":                "ranged",
	"a leather sling":                     "ranged",
	"a heavy throwing dart":               "weapon",
	"a black longsword of destruction":    "weapon",
	"the infernal stiletto of bane":       "weapon",
	"a wooden druid's staff":              "staff",
	"a suit of lordly plate":              "armor",
	"a jug of water":                      "drink",
	"a tiny kitten":                       "other",
	"a tiny mithril stud set with a ruby": "worn",
	"a tiny mithril stud":                 "worn",
}

// nyyStats are the short_stats that aren't Kegor's.
var nyyStats = map[string]string{
	"a scarlet potion": "a scarlet potion * (Potion) Lvl:41 harm - heal " +
		"* Wt:1 Val:1.5p * Ribcage",
	"a rolled scroll": "a rolled scroll * (Scroll) Lvl:50 bless - armor - " +
		"remove curse * Wt:2 Val:5p * Mithril Hall (S)",
	"a vial of tree frog toxin": "a vial of tree frog toxin * (Poison) " +
		"Lvl:21 Type:fumble Apps:5 Hits:3 * Wt:1 Val:10p * Skullport (S)",
	"a vial of black adder venom": "a vial of black adder venom * (Poison) " +
		"Lvl:40 Type:enervate Apps:1 * Wt:1 Val:2p * Skullport (S)",
	"a murky potion": "a murky potion",
}

// nyyLabels are the labels NyyLIB shows for potions, scrolls and poisons in
// bags.
var nyyLabels = map[string]string{
	"a scarlet potion":            "potion Lvl:41 harm - heal",
	"a rolled scroll":             "scroll Lvl:50 bless - armor - remove curse",
	"a vial of tree frog toxin":   "poison Lvl:21 Type:fumble Apps:5 Hits:3",
	"a vial of black adder venom": "poison Lvl:40 Type:enervate Apps:1",
}

// nyyFlags are the item_flags of the items that have any.
var nyyFlags = map[string]string{
	"a scarlet potion":                 "magic",
	"an ironwood bow":                  "two_hand",
	"a well-made crossbow":             "two_hand",
	"a black longsword of destruction": "float magic no_burn no_loc two_hand",
	"the infernal stiletto of bane":    "float magic no_burn no_loc",
	"a wooden druid's staff":           "bless hidden no_drop two_hand",
	"a suit of lordly plate":           "magic no_rent whole_body",
}

// nyyButtons are the weapon buttons NyyLIB sets up for a non-ranger who
// wields each weapon.
var nyyButtons = map[string]string{
	"an ironwood bow":                  "Bow",
	"a well-made crossbow":             "Bow",
	"a leather sling":                  "Bow",
	"a heavy throwing dart":            "Shield",
	"a black longsword of destruction": "TwoHand",
	"the infernal stiletto of bane":    "Shield",
}

// nyyTags are NyyLIB's patterns for the labels, "%(Potion%)([A-Za-z0-9:
// -]+)%*" and the like, by item_type. It reads one wherever short_stats has
// "(Potion) ", and stops with an error if the pattern then doesn't match.
// Go finds the same match as Lua here: the class can't match the '*'.
var nyyTags = map[string]string{
	"potion": "(Potion)", "scroll": "(Scroll)", "poison": "(Poison)",
}

// oldItems are toril.db's item tables as they were when gobot read them,
// before ee90880, with an item and some details of it.
const oldItems = `CREATE TABLE items(
	item_id INTEGER PRIMARY KEY
	,item_name VARCHAR(100) NOT NULL
	,keywords VARCHAR(100) NOT NULL
	,weight INTEGER
	,c_value INTEGER
	,item_type VARCHAR(10) REFERENCES item_types(type_abbr) NOT NULL
	,from_zone VARCHAR(25) REFERENCES zones(zone_abbr) NOT NULL
	,from_mob VARCHAR(150) REFERENCES mobs(mob_name)
	,short_stats VARCHAR(500)
	,long_stats VARCHAR(800)
	,full_stats TEXT
	,comments TEXT
	,last_id DATE
);
CREATE TABLE item_flags(
	item_id INTEGER REFERENCES items(item_id)
	,flag_abbr VARCHAR(10) REFERENCES flags(flag_abbr)
	,PRIMARY KEY (item_id, flag_abbr)
);
CREATE TABLE item_slots(
	item_id INTEGER REFERENCES items(item_id)
	,slot_abbr VARCHAR(10) REFERENCES slots(slot_abbr)
	,PRIMARY KEY (item_id, slot_abbr)
);
CREATE TABLE item_attribs(
	item_id INTEGER REFERENCES items(item_id)
	,attrib_abbr VARCHAR(10) REFERENCES attribs(attrib_abbr)
	,attrib_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, attrib_abbr)
);
INSERT INTO items VALUES (3, 'an old sword', 'old sword', 5, 100, 'weapon',
	'WD', NULL, 'an old sword (Wielded) * 2D4 * Zone: WD * Last ID: 2010-01-01',
	NULL, NULL, NULL, '2010-01-01');
INSERT INTO item_flags VALUES (3, 'two_hand');
INSERT INTO item_slots VALUES (3, 'wield');
INSERT INTO item_attribs VALUES (3, 'hit', 2);
`

// people are a char, a hidden alt and an account with its player's name,
// which dbcopy.sh mustn't publish.
const people = `INSERT INTO accounts VALUES ('yog', 'Yogi Bear');
INSERT INTO chars VALUES ('yog', 'Yog', 'Shaman', 'Barbarian', 50,
	'2026-11-01 12:00:00', 't');
INSERT INTO chars VALUES ('yog', 'Hid', 'Rogue', 'Human', 50,
	'2026-11-01 12:00:00', 'f');
`

// makeTorilDB makes a toril.db at path with init_db.sql, then runs more in
// it.
func makeTorilDB(t *testing.T, path, more string) {
	t.Helper()
	src, err := os.Getwd()
	chkErr(t, err)
	out, err := exec.Command("sqlite3", "-bail", path,
		".read "+strconv.Quote(filepath.Join(src, "init_db.sql")),
		more).CombinedOutput()
	if err != nil {
		t.Fatalf("making %s: %v\n%s", path, err, out)
	}
}

// refillItems runs dbcopy_items.sql on the toril.db at path as dbcopy.sh
// does, with the TorilDB.db at kegor attached read-only.
func refillItems(t *testing.T, path, kegor string) (string, error) {
	t.Helper()
	src, err := os.Getwd()
	chkErr(t, err)
	out, err := exec.Command("sqlite3", "-bail", path,
		"ATTACH 'file:"+filepath.ToSlash(kegor)+"?mode=ro' AS k",
		".read "+strconv.Quote(filepath.Join(src, "dbcopy_items.sql")),
	).CombinedOutput()
	return string(out), err
}

// openRO opens the database at path read-only.
func openRO(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(path)+"?mode=ro")
	chkErr(t, err)
	return db
}

// queryInt runs a query for one number.
func queryInt(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Errorf("%s: %v", query, err)
	}
	return n
}

// weaponButton is NyyLIB's choice of weapon buttons for a non-ranger who
// wields name, by its own three queries: Bow, TwoHand, Shield, or "" for
// none.
func weaponButton(t *testing.T, db *sql.DB, name string) string {
	t.Helper()
	like := "item_name LIKE '%" + strings.ReplaceAll(name, "'", "''") + "%'"
	for _, q := range []struct{ button, query string }{
		{"Bow", "SELECT item_name FROM items WHERE item_type = 'ranged' " +
			"AND " + like},
		{"TwoHand", "SELECT item_name FROM items, item_flags WHERE " +
			"item_type = 'weapon' AND " + like + " AND item_flags.item_id = " +
			"items.item_id AND item_flags.flag_abbr = 'two_hand'"},
		{"Shield", "SELECT item_name FROM items WHERE (item_type = 'weapon' " +
			"or item_type = \"ranged\") AND " + like},
	} {
		var got string
		switch err := db.QueryRow(q.query).Scan(&got); err {
		case nil:
			return q.button
		case sql.ErrNoRows:
		default:
			t.Error(err)
		}
	}
	return ""
}

// checkNyyLIB checks the item tables of the toril.db at path, refilled from
// nyyEq, as NyyLIB reads them.
func checkNyyLIB(t *testing.T, what, path string) {
	t.Helper()
	db := openRO(t, path)
	defer db.Close()

	// a script of NyyLIB's own inserts all 13 columns by position
	rows, err := db.Query("SELECT name FROM pragma_table_info('items')")
	chkErr(t, err)
	var cols []string
	for rows.Next() {
		var c string
		chkErr(t, rows.Scan(&c))
		cols = append(cols, c)
	}
	ChkRows(rows)
	if got, want := strings.Join(cols, " "), "item_id item_name keywords "+
		"weight c_value item_type from_zone from_mob short_stats long_stats "+
		"full_stats comments last_id"; got != want {
		t.Errorf("%s: items columns: %s", what, got)
	}

	// every item, its type and stats, and the labels NyyLIB reads from them
	// when it starts
	kegor := make(map[string]string)
	for _, r := range nyyEq {
		s, _ := r["short_stats"].(string)
		kegor[r["name"].(string)] = s
	}
	labels := make(map[string]string)
	rows, err = db.Query("SELECT item_name, item_type, short_stats FROM items")
	chkErr(t, err)
	n := 0
	for rows.Next() {
		var name, typ string
		var stats sql.NullString
		chkErr(t, rows.Scan(&name, &typ, &stats))
		n++
		if typ != nyyTypes[name] {
			t.Errorf("%s: %s: type %q, want %q", what, name, typ,
				nyyTypes[name])
		}
		want, ok := nyyStats[name]
		if !ok {
			want = kegor[name]
		}
		if !stats.Valid || stats.String != want {
			t.Errorf("%s: %s: short_stats %q, want %q", what, name,
				stats.String, want)
		}
		tag, ok := nyyTags[typ]
		if !ok || !strings.Contains(stats.String, tag+" ") {
			continue
		}
		m := regexp.MustCompile(regexp.QuoteMeta(tag) +
			`([A-Za-z0-9: -]+)\*`).FindStringSubmatch(stats.String)
		if m == nil {
			t.Errorf("%s: %s would stop NyyLIB: %q", what, name, stats.String)
			continue
		}
		labels[name] = typ + " " + strings.TrimSpace(m[1])
	}
	ChkRows(rows)
	if n != len(nyyEq) {
		t.Errorf("%s: %d items, want %d", what, n, len(nyyEq))
	}
	chkMap(t, what+": labels", nyyLabels, labels)

	flags := make(map[string]string)
	rows, err = db.Query("SELECT item_name, flag_abbr FROM items " +
		"JOIN item_flags USING (item_id)")
	chkErr(t, err)
	for rows.Next() {
		var name, flag string
		chkErr(t, rows.Scan(&name, &flag))
		flags[name] = strings.TrimSpace(flags[name] + " " + flag)
	}
	ChkRows(rows)
	for name, f := range flags {
		list := strings.Fields(f)
		sort.Strings(list)
		flags[name] = strings.Join(list, " ")
	}
	chkMap(t, what+": flags", nyyFlags, flags)

	for name, want := range nyyButtons {
		if got := weaponButton(t, db, name); got != want {
			t.Errorf("%s: %s: %q buttons, want %q", what, name, got, want)
		}
	}

	// the shortest name comes first
	var first string
	chkErr(t, db.QueryRow("SELECT item_name FROM items WHERE "+
		"item_name LIKE '%a tiny mithril stud%'").Scan(&first))
	if first != "a tiny mithril stud" {
		t.Errorf("%s: first match: %s", what, first)
	}

	// last_id || '': the date as stored, which the driver would otherwise
	// turn into a time, the column being a DATE
	var kw, zone, date string
	var wt, val int
	chkErr(t, db.QueryRow("SELECT keywords, weight, c_value, from_zone, "+
		"last_id || '' FROM items WHERE item_name = 'a scarlet potion'").Scan(
		&kw, &wt, &val, &zone, &date))
	if kw != "scarlet potion" || wt != 1 || val != 1500 || zone != "Ribcage" ||
		date != "2024-03-01" {
		t.Errorf("%s: a scarlet potion: %q %d %d %q %q", what, kw, wt, val,
			zone, date)
	}
	chkErr(t, db.QueryRow("SELECT keywords, from_zone FROM items WHERE "+
		"item_name = 'a tiny kitten'").Scan(&kw, &zone))
	if kw != "" || zone != "" {
		t.Errorf("%s: a tiny kitten: keywords %q, zone %q", what, kw, zone)
	}
	if n := queryInt(t, db, "SELECT count(*) FROM sqlite_master WHERE "+
		"name IN ('item_slots', 'item_attribs')"); n != 0 {
		t.Errorf("%s: %d tables of old item details left", what, n)
	}
}

// chkMap checks that got has what want has, and no more.
func chkMap(t *testing.T, what string, want, got map[string]string) {
	t.Helper()
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %s: %q, want %q", what, k, got[k], v)
		}
	}
	for k, v := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s: %s: %q, want none", what, k, v)
		}
	}
}

// chkOldItems checks that the toril.db at path still has oldItems.
func chkOldItems(t *testing.T, what, path string) {
	t.Helper()
	db := openRO(t, path)
	defer db.Close()
	var name string
	chkErr(t, db.QueryRow("SELECT group_concat(item_name) FROM items").Scan(
		&name))
	if name != "an old sword" ||
		queryInt(t, db, "SELECT count(*) FROM item_flags") != 1 ||
		queryInt(t, db, "SELECT count(*) FROM item_slots") != 1 {
		t.Errorf("%s: the old item tables changed: items %q", what, name)
	}
}

// writeEq makes a TorilDB.db at path whose eq table is made by schema.
func writeEq(t *testing.T, path, schema string) {
	t.Helper()
	os.Remove(path)
	db, err := sql.Open("sqlite3", path)
	chkErr(t, err)
	defer db.Close()
	_, err = db.Exec(schema)
	chkErr(t, err)
}

// Test_DBCopyItems runs dbcopy_items.sql as dbcopy.sh does: on a toril.db
// made by init_db.sql, which has no item tables, and on one with the old
// ones. When it fails, nothing changes.
func Test_DBCopyItems(t *testing.T) {
	dir := t.TempDir()
	kegor := filepath.Join(dir, "TorilDB.db")
	writeItemDB(t, kegor, nyyEq)

	for _, c := range []struct{ what, more string }{
		{"made by init_db.sql", ""},
		{"with the old item tables", oldItems},
	} {
		path := filepath.Join(dir, "toril.db")
		os.Remove(path)
		makeTorilDB(t, path, c.more)
		if out, err := refillItems(t, path, kegor); err != nil || out != "" {
			t.Fatalf("%s: %v %s", c.what, err, out)
		}
		checkNyyLIB(t, c.what, path)
	}

	none := filepath.Join(dir, "none.db")
	cols := filepath.Join(dir, "cols.db")
	writeEq(t, cols, "CREATE TABLE eq (name VARCHAR PRIMARY KEY, "+
		"short_stats VARCHAR)")
	// Kegor's text with a "(Potion) " NyyLIB couldn't read: it can't be
	// put back as it was
	stopper := filepath.Join(dir, "stopper.db")
	writeItemDB(t, stopper, []dbRow{{"name": "a strange potion",
		"type": "POTION", "short_stats": "a strange potion * (Potion) " +
			"Lvl:1 ? * Wt:1 Val:0 * Ako Village"}})
	for _, c := range []struct{ what, kegor, err string }{
		{"no TorilDB.db", none, "unable to open database"},
		{"a TorilDB.db without the columns", cols, "no such column"},
		{"a row that would stop NyyLIB", stopper,
			"CHECK constraint failed: nyylib_would_not_start"},
	} {
		path := filepath.Join(dir, "toril.db")
		os.Remove(path)
		makeTorilDB(t, path, oldItems)
		if out, err := refillItems(t, path, c.kegor); err == nil ||
			!strings.Contains(out, c.err) {
			t.Errorf("%s: %v %s", c.what, err, out)
		}
		chkOldItems(t, c.what, path)
	}
	if _, err := os.Stat(none); !os.IsNotExist(err) {
		t.Errorf("no TorilDB.db: one was made: %v", err)
	}
}

// findBash finds bash to run dbcopy.sh with. On Windows that's Git for
// Windows' bash, which can take paths like C:/x and run the sqlite3 on the
// PATH.
func findBash(t *testing.T) string {
	if runtime.GOOS == "windows" {
		bash := filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin",
			"bash.exe")
		if _, err := os.Stat(bash); err != nil {
			t.Skip("dbcopy.sh needs Git for Windows' bash")
		}
		return bash
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("dbcopy.sh needs bash")
	}
	return bash
}

// Test_DBCopy runs dbcopy.sh. What it publishes has no hidden alts or real
// names, and NyyLIB's items from the TorilDB.db beside the source, or the
// item tables it had if that's missing or can't be used, with the reason
// on stderr.
func Test_DBCopy(t *testing.T) {
	bash := findBash(t)
	src, err := os.Getwd()
	chkErr(t, err)
	dir := t.TempDir()
	tmp := filepath.Join(dir, "tmp")
	for _, d := range []string{"bot", "pub", "tmp"} {
		chkErr(t, os.Mkdir(filepath.Join(dir, d), 0750))
	}
	bot := filepath.Join(dir, "bot", "toril.db")
	makeTorilDB(t, bot, oldItems+people)
	kegor := filepath.Join(dir, "bot", "TorilDB.db")
	writeItemDB(t, kegor, nyyEq)
	pub := filepath.Join(dir, "pub", "toril.db")

	// run runs dbcopy.sh, with its snapshot in tmp, and returns its stderr.
	// Paths go to bash as C:/x on Windows, which it and sqlite3 both take.
	// TMPDIR is set inside bash, since Git for Windows' bash turns a C:/x
	// in the environment it starts with into /c/x, which sqlite3 can't open.
	run := func(what string) string {
		t.Helper()
		cmd := exec.Command(bash, "-c", `export TMPDIR="$1"; shift; `+
			`exec bash "$@"`, "bash", filepath.ToSlash(tmp),
			filepath.ToSlash(filepath.Join(src, "dbcopy.sh")),
			filepath.ToSlash(bot), filepath.ToSlash(pub))
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s: dbcopy.sh: %v\n%s", what, err, stderr.String())
		}
		if stdout.Len() > 0 {
			t.Errorf("%s: dbcopy.sh printed %q", what, stdout.String())
		}
		left, err := os.ReadDir(tmp)
		chkErr(t, err)
		if len(left) > 0 {
			t.Errorf("%s: dbcopy.sh left %s in TMPDIR", what, left[0].Name())
		}
		if _, err := os.Stat(pub + ".new"); !os.IsNotExist(err) {
			t.Errorf("%s: dbcopy.sh left toril.db.new: %v", what, err)
		}

		db := openRO(t, pub)
		defer db.Close()
		for _, c := range []struct {
			query string
			want  int
		}{
			{"SELECT count(*) FROM chars WHERE vis = 'f'", 0},
			{"SELECT count(*) FROM chars", 1},
			{"SELECT count(*) FROM accounts WHERE player_name IS NOT NULL", 0},
			// VACUUMed, so the deleted rows aren't in the file
			{"PRAGMA freelist_count", 0},
		} {
			if got := queryInt(t, db, c.query); got != c.want {
				t.Errorf("%s: %s: %d, want %d", what, c.query, got, c.want)
			}
		}
		return stderr.String()
	}

	if msg := run("refilled"); msg != "" {
		t.Errorf("refilled: dbcopy.sh said %q", msg)
	}
	checkNyyLIB(t, "published", pub)

	chkErr(t, os.Remove(kegor))
	if msg := run("no TorilDB.db"); !strings.Contains(msg,
		"TorilDB.db, so the item tables aren't refreshed") {
		t.Errorf("no TorilDB.db: dbcopy.sh said %q", msg)
	}
	chkOldItems(t, "no TorilDB.db", pub)
	if _, err := os.Stat(kegor); !os.IsNotExist(err) {
		t.Errorf("no TorilDB.db: one was made: %v", err)
	}

	writeEq(t, kegor, "CREATE TABLE eq (name VARCHAR PRIMARY KEY, "+
		"short_stats VARCHAR)")
	if msg := run("a TorilDB.db it can't use"); !strings.Contains(msg,
		"no such column") || !strings.Contains(msg,
		"the item tables weren't refreshed from") {
		t.Errorf("a TorilDB.db it can't use: dbcopy.sh said %q", msg)
	}
	chkOldItems(t, "a TorilDB.db it can't use", pub)
}
