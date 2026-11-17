package main

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// mobsSchema is the mobs table as Kegor's TorilDB.db has it.
const mobsSchema = "CREATE TABLE mobs (name VARCHAR (120) PRIMARY KEY " +
	"NOT NULL UNIQUE, keywords VARCHAR (120), size VARCHAR (20), " +
	"level NUMERIC (6), race VARCHAR (50), rank VARCHAR (50), " +
	"priority NUMERIC (8), lock VARCHAR (5), aggro VARCHAR (10), " +
	"shield VARCHAR (10), track VARCHAR (10), charm VARCHAR (10), " +
	"hidden VARCHAR (10), ally VARCHAR (10), fight VARCHAR (10), " +
	"zone VARCHAR (120), load VARCHAR (20), equipment VARCHAR (1000), " +
	"quests VARCHAR (1000), shop VARCHAR (10), guild VARCHAR (20), " +
	"long_name VARCHAR (200), long_name_2 VARCHAR (200), " +
	"long_name_3 VARCHAR (200), long_name_4 VARCHAR (200), " +
	"long_name_5 VARCHAR (200), long_name_6 VARCHAR (200), " +
	"long_name_7 VARCHAR (200), long_name_8 VARCHAR (200), " +
	"short_stats VARCHAR (4000), long_stats VARCHAR (4000))"

// Stats text for the test mobs, in Kegor's format.
const (
	fiendShort = "the pit fiend, Kri'ik (Kri'ik) * QUEST-MOB * SHIELD * " +
		"Level 59 Baatezu (Massive) * Avernus - Undermountain"
	giantShort = "a fire giant (fire giant) * AGGRO * Level 52 Giant " +
		"(Huge) * Jotunheim"
	jewelerShort = "a duergar jeweler (duergar jeweler) * Level 30 " +
		"Duergar (Small) * Skullport"
	bartender1 = "the bartender 1 (bartender) * Level 20 Human (Medium) " +
		"* Waterdeep"
	bartender2 = "the bartender 2 (bartender) * Level 20 Human (Medium) " +
		"* Calimport"
	bartender10 = "the bartender 10 (bartender) * Level 25 Dwarf (Medium) " +
		"* Mithril Hall"
)

// testMobs is the test TorilDB.db's mobs table.
var testMobs = append([]dbRow{
	{"name": "the pit fiend, Kri'ik", "keywords": "Kri'ik",
		"zone":      "Avernus - Undermountain",
		"equipment": "a black longsword of destruction|some demonscale boots",
		"quests": "a pendant with the crystal missing|" +
			"the mark of the dragonhunter",
		"short_stats": fiendShort, "long_stats": fiendShort + " * Prio:1"},
	// Kegor's eq lists don't always match an item's name in case
	{"name": "a fire giant", "zone": "Jotunheim",
		"equipment":   "A Black Longsword Of Destruction",
		"short_stats": giantShort, "long_stats": giantShort},
	{"name": "a duergar jeweler", "zone": "Skullport",
		"equipment":   "a tiny mithril stud set with a ruby",
		"short_stats": jewelerShort, "long_stats": jewelerShort},
	// Kegor numbers mobs that share a name. Most carry nothing he lists.
	{"name": "the bartender 1", "zone": "Waterdeep",
		"equipment": "a tiny mithril stud", "short_stats": bartender1,
		"long_stats": bartender1},
	{"name": "the bartender 2", "zone": "Calimport",
		"short_stats": bartender2, "long_stats": bartender2},
	{"name": "the bartender 10", "zone": "Mithril Hall",
		"short_stats": bartender10, "long_stats": bartender10},
	// would end the tell and run a command if passed along as is
	{"name": "an evil imp", "zone": "Waterdeep",
		"equipment":   "a ring;#showme pwned|{x}",
		"short_stats": "an evil imp\n#showme pwned; say $cpw @x {y} \\z",
		"long_stats":  "an evil imp"},
}, drowAssassins()...)

// assassinShort is Kegor's short stats for the test drow assassin
// numbered n.
func assassinShort(n int) string {
	return fmt.Sprintf("a drow assassin %d (drow assassin) * AGGRO * "+
		"Level 45 Drow Elf (Medium) * Menzoberranzan", n)
}

// drowAssassins are 12 mobs sharing a name, more than mstat lists, and all
// carrying the stiletto, more than astat's Loads on line names.
func drowAssassins() []dbRow {
	var mobs []dbRow
	for n := 1; n <= 12; n++ {
		mobs = append(mobs, dbRow{
			"name": fmt.Sprintf("a drow assassin %d", n),
			"zone": "Menzoberranzan", "equipment": "the infernal stiletto of bane",
			"short_stats": assassinShort(n), "long_stats": assassinShort(n)})
	}
	return mobs
}

func testMstat(t *testing.T) {
	bartenders := append(append(tell(bartender1+" * Eq: a tiny mithril stud"),
		tell(bartender2)...), tell(bartender10)...)
	var assassins []string
	for n := 1; n <= mobLimit; n++ {
		assassins = append(assassins, tell(assassinShort(n)+
			" * Eq: the infernal stiletto of bane")...)
	}
	assassins = append(assassins, tell("...and 2 more mobs by that name.")...)
	checks := []struct {
		tell string
		good []string
	}{
		// Kegor's stats, then the eq the mob carries and its quest's items
		{"mstat kri'ik", tell(fiendShort + " * Eq: a black longsword of " +
			"destruction, some demonscale boots * Quest: a pendant with the " +
			"crystal missing, the mark of the dragonhunter")},
		{"mstat fire giant", tell(giantShort +
			" * Eq: A Black Longsword Of Destruction")},
		// every mob sharing the name, 2 before 10
		{"mstat the bartender", bartenders},
		{"mstat bartender", bartenders},
		// the whole name, number and all, is just that mob
		{"mstat The Bartender 10", tell(bartender10)},
		// the closest match first, then the mobs sharing its name
		{"mstat bartender 10", append(append(tell(bartender10),
			tell(bartender1+" * Eq: a tiny mithril stud")...),
			tell(bartender2)...)},
		// no more than mobLimit of them, and a count of the rest
		{"mstat drow assassin", assassins},
		{"mstat evil imp", tell("an evil imp #showme pwned, say cpw x (y) z " +
			"* Eq: a ring,#showme pwned, (x)")},
		{"mstat blah", []string{"t Yog 404 mob not found: blah\n"}},
		{"mstat", []string{"t Yog " + BadSyntax + "\n",
			"t Yog See 'tell katumi help mstat'.\n"}},
	}
	for _, c := range checks {
		chk(t, "ReplyTo(Yog, "+c.tell+")", c.good, ReplyTo("Yog", c.tell))
	}
}

// zoneItems are items Kegor numbers, with their zones: four short swords,
// one loading in two zones and two in the same one, a rusty sword with two
// numbered namesakes, one where it loads too, and a third stiletto, where
// the drow assassins carry the first.
var zoneItems = [][2]string{
	{"a short sword 1", "Waterdeep"},
	{"a short sword 2", "Silverymoon - Westgate"},
	{"a short sword 3", "Calimport"},
	{"a short sword 4", "Calimport"},
	{"a rusty sword", "Undermountain"},
	{"a rusty sword 1", "West Falls"},
	{"a rusty sword 2", "Undermountain"},
	{"the infernal stiletto of bane 3", "Menzoberranzan"},
}

// zoneLong is the long stats text of one of zoneItems.
func zoneLong(name, zone string) string {
	return name + " * WEAPON (WIELD) * " + zone
}

// zoneMobs carry zoneItems, mostly by the name the game shows, without
// Kegor's number, as his eq lists do.
var zoneMobs = []dbRow{
	{"name": "a city guard 1", "zone": "Waterdeep",
		"equipment": "a short sword|a wooden shield"},
	// names and zones in any case
	{"name": "a city guard 2", "zone": "WestGate",
		"equipment": "A Short Sword"},
	{"name": "a caravan master", "zone": "Scornubel - Westgate",
		"equipment": "a short sword"},
	// where two short swords load, or none
	{"name": "a city guard 3", "zone": "Calimport",
		"equipment": "a short sword"},
	{"name": "a wandering minstrel", "zone": "Silverymoon - Waterdeep",
		"equipment": "a short sword"},
	{"name": "a pirate", "zone": "Nhavan Island",
		"equipment": "a short sword"},
	// by the whole name, once with the name without the number as well
	{"name": "the armorer", "zone": "Waterdeep",
		"equipment": "a short sword 1|a short sword"},
	{"name": "a weaponsmith", "zone": "Calimport",
		"equipment": "a short sword 3"},
	{"name": "the zombie lord", "zone": "West Falls",
		"equipment": "a rusty sword"},
	{"name": "a robust orc", "zone": "Undermountain",
		"equipment": "a rusty sword"},
}

// testLoadsOnZones checks that astat's Loads on line for an item Kegor
// numbers also names the mobs whose eq lists give the name without his
// number, where their zones say it's that item they carry.
func testLoadsOnZones(t *testing.T) {
	items := append([]dbRow{}, testEq...)
	for _, it := range zoneItems {
		items = append(items, dbRow{"name": it[0], "zone": it[1],
			"type": "WEAPON", "worn": "WIELD", "date": "2024-04-04",
			"short_stats": it[0] + " (WIELD) * " + it[1],
			"long_stats":  zoneLong(it[0], it[1])})
	}
	writeItemDB(t, itemDBFile, items)
	defer writeItemDB(t, itemDBFile, testEq)
	db, err := sql.Open("sqlite3", itemDBFile)
	chkErr(t, err)
	insertRows(t, db, "mobs", zoneMobs)
	chkErr(t, db.Close())

	// astat's lines for an item: its long stats, then the mobs, if any
	astat := func(name, zone, mobs string) []string {
		if mobs == "" {
			return tell(zoneLong(name, zone))
		}
		return append(tell(zoneLong(name, zone)), tell("Loads on: "+mobs)...)
	}
	sword2 := astat("a short sword 2", "Silverymoon - Westgate",
		"a caravan master (Scornubel - Westgate), a city guard 2 (WestGate)")
	var assassins []string
	for n := 1; n <= loadsOnLimit; n++ {
		assassins = append(assassins,
			fmt.Sprintf("a drow assassin %d (Menzoberranzan)", n))
	}
	checks := []struct {
		tell string
		good [][]string
	}{
		// the mobs that name an item whole come first, then those that name
		// it without the number, in a zone where no other item of that name
		// loads. A zone of two parts shares either one.
		{"astat short sword", [][]string{
			astat("a short sword 1", "Waterdeep",
				"the armorer (Waterdeep), a city guard 1 (Waterdeep)"),
			sword2,
			astat("a short sword 3", "Calimport", "a weaponsmith (Calimport)"),
			astat("a short sword 4", "Calimport", "")}},
		// the one a number names
		{"astat a short sword 2", [][]string{sword2}},
		// an item without a number keeps to the mobs that name it; one with
		// a number gets those in its zone, unless the other loads there too
		{"astat rusty sword", [][]string{
			astat("a rusty sword", "Undermountain", "a robust orc "+
				"(Undermountain), the zombie lord (West Falls)"),
			astat("a rusty sword 1", "West Falls",
				"the zombie lord (West Falls)"),
			astat("a rusty sword 2", "Undermountain", "")}},
		// no more than loadsOnLimit of them, and a count of the rest
		{"astat the infernal stiletto of bane 3", [][]string{astat(
			"the infernal stiletto of bane 3", "Menzoberranzan",
			strings.Join(assassins, ", ")+", and 7 more mobs")}},
	}
	for _, c := range checks {
		var good []string
		for _, lines := range c.good {
			good = append(good, lines...)
		}
		chk(t, "ReplyTo(Yog, "+c.tell+")", good, ReplyTo("Yog", c.tell))
	}
	// -item, which Ynnd's ASSOC stat runs, has no Loads on line
	chk(t, "FindItem(a short sword 2)",
		[]string{"a short sword 2 (WIELD) * Silverymoon - Westgate"},
		FindItem("a short sword 2", "short_stats"))
}

// testMobsDown checks that a TorilDB.db without a mobs table gobot can read
// gets mstat a reply saying so, and costs astat only its Loads on line.
func testMobsDown(t *testing.T) {
	down := []string{"t Yog " + mobsDown + "\n"}
	for what, stmt := range map[string]string{
		"no mobs table":     "DROP TABLE mobs",
		"equipment renamed": "ALTER TABLE mobs RENAME COLUMN equipment TO eq",
	} {
		changeItemDB(t, itemDBFile, stmt)
		chk(t, "TorilDB.db "+what+": mstat", down,
			ReplyTo("Yog", "mstat kri'ik"))
		chk(t, "TorilDB.db "+what+": astat", tell(swordLong),
			ReplyTo("Yog", "astat destruction sword"))
	}
	writeItemDB(t, itemDBFile, testEq)
}
