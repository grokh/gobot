package main

import (
	"fmt"
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
