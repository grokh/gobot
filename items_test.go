package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// eqSchema is the eq table as Kegor's TorilDB.db has it.
const eqSchema = "CREATE TABLE eq (name VARCHAR PRIMARY KEY UNIQUE NOT NULL, " +
	"keywords VARCHAR, zone VARCHAR, load VARCHAR, quest VARCHAR, " +
	"no_id VARCHAR, bound VARCHAR, type VARCHAR, worn VARCHAR, wt INT (10), " +
	"value INT (10), ac INT (10), armor INT (10), pages INT (10), " +
	"hp INT (10), p_hp INT (10), w_dice_count INT (10), w_dice INT (10), " +
	"w_type VARCHAR, w_class VARCHAR, w_range VARCHAR, w_bonus VARCHAR, " +
	"hit INT (10), dam INT (10), sv_spell INT (10), sv_bre INT (10), " +
	"sv_para INT (10), sv_petri INT (10), sv_rod INT (10), str INT (10), " +
	"agi INT (10), dex INT (10), con INT (10), pow INT (10), int INT (10), " +
	"wis INT (10), cha INT (10), max_str INT (10), max_agi INT (10), " +
	"max_dex INT (10), max_con INT (10), max_pow INT (10), " +
	"max_int INT (10), max_wis INT (10), max_cha INT (10), luck INT (10), " +
	"karma INT (10), mana INT (10), move INT (10), age INT (10), " +
	"weight INT (10), height INT (10), mr INT (10), sf_ele INT (10), " +
	"sf_enc INT (10), sf_heal INT (10), sf_ill INT (10), sf_inv INT (10), " +
	"sf_nat INT (10), sf_nec INT (10), sf_prot INT (10), sf_spi INT (10), " +
	"sf_sum INT (10), psp INT (10), i_type VARCHAR, i_quality INT (10), " +
	"i_stutter INT (10), i_min INT (10), holds INT (10), " +
	"weightless INT (10), pick INT (10), break INT (10), p_poison VARCHAR, " +
	"p_level INT (10), p_apps INT (10), p_hits INT (10), charge INT (10), " +
	"max_charge INT (10), s_level INT (10), s_spell VARCHAR, " +
	"enchant VARCHAR, crit VARCHAR, bonus VARCHAR, effects VARCHAR, " +
	"called VARCHAR, powers VARCHAR, gearset VARCHAR, r_unarmd INT (10), " +
	"r_slash INT (10), r_bludgn INT (10), r_pierce INT (10), " +
	"r_ranged INT (10), r_spell INT (10), r_sonic INT (10), r_pos INT (10), " +
	"r_neg INT (10), r_psi INT (10), r_mental INT (10), r_good INT (10), " +
	"r_evil INT (10), r_law INT (10), r_chaos INT (10), r_force INT (10), " +
	"r_fire INT (10), r_cold INT (10), r_elect INT (10), r_acid INT (10), " +
	"r_poison INT (10), affect_flags VARCHAR, item_flags VARCHAR (5000), " +
	"usable_by VARCHAR, date VARCHAR, short_stats VARCHAR (5000), " +
	"long_stats VARCHAR (5000))"

// dbRow is one item or mob for a test TorilDB.db, by column.
type dbRow map[string]interface{}

// writeItemDB makes a TorilDB.db at path holding items, with testMobs in
// its mobs table.
func writeItemDB(t *testing.T, path string, items []dbRow) {
	os.Remove(path)
	db, err := sql.Open("sqlite3", path)
	chkErr(t, err)
	defer db.Close()
	for _, schema := range []string{eqSchema, mobsSchema} {
		_, err = db.Exec(schema)
		chkErr(t, err)
	}
	insertRows(t, db, "eq", items)
	insertRows(t, db, "mobs", testMobs)
}

// insertRows adds rows to a table of a test TorilDB.db.
func insertRows(t *testing.T, db *sql.DB, table string, rows []dbRow) {
	for _, row := range rows {
		var cols, marks []string
		var args []interface{}
		for c, v := range row {
			cols = append(cols, col(c))
			marks = append(marks, "?")
			args = append(args, v)
		}
		_, err := db.Exec("INSERT INTO "+table+" ("+strings.Join(cols, ", ")+
			") VALUES ("+strings.Join(marks, ", ")+")", args...)
		chkErr(t, err)
	}
}

// Stats text for the test items, in Kegor's formats.
const (
	stilettoShort = "the infernal stiletto of bane (WIELD) * 4D4 5/4 " +
		"6%/2x * HASTE SLOW-POISON * THIEF-ONLY * Wt:5 Val:0 * Tiamat (R)"
	stilettoLong = "the infernal stiletto of bane (infernal stiletto bane) " +
		"* WEAPON (WIELD) * Wt:5 Val:0 * Type:Dagger Class:Simple Dice:4D4 " +
		"Hit:5 Dam:4 CritRange:6% CritBonus:2x * HASTE SLOW-POISON * MAGIC " +
		"FLOAT NOBURN NOLOCATE NO-WARRIOR NO-CLERIC NO-MAGE * Tiamat (R) " +
		"* 2021-11-13"
	rubyShort = "a tiny mithril stud set with a ruby (EAR) * 0/3 MaxAgi:3 " +
		"* Fire:5% * Wt:0 Val:501p * Skullport (Q)"
	rubyLong = "a tiny mithril stud set with a ruby (stud mithril set ruby) " +
		"* WORN (EARRING) * Wt:0 Val:501000 * Dam:3 MaxAgi:3 * Fire:5% " +
		"* NOBURN * Skullport (Q) * 2025-01-16"
	studShort = "a tiny mithril stud (EAR) * 0/1 * Wt:0 Val:1p * Skullport (Q)"
	studLong  = "a tiny mithril stud (stud mithril) * WORN (EARRING) * Wt:0 " +
		"Val:1000 * Dam:1 * Skullport (Q) * 2026-11-01"
	swordShort = "a black longsword of destruction (WIELD 2H) * 8D6 5/8 " +
		"6%/2x * Fire:5% * INFRA * Proc: Battle Rage * " +
		"BLACKGUARD/HEXBLADE/WARRIOR-ONLY * Wt:15 Val:10p * QUEST-ITEM " +
		"* Jotunheim (I)"
	swordLong = "a black longsword of destruction (black sword destruction " +
		"twilight) * WEAPON (WIELD 2H) * Wt:15 Val:10000 * Type:Longsword " +
		"Class:Martial Dice:8D6 Hit:5 Dam:8 CritRange:6% CritBonus:2x " +
		"* Fire:5% * INFRA * Proc: Battle Rage * MAGIC FLOAT NOBURN " +
		"NOLOCATE NO-CLERIC NO-THIEF NO-MAGE ANTI-RANGER ANTI-PALADIN " +
		"* QUEST-ITEM * Jotunheim (I) * 2025-08-30"
	markShort = `the mark of the dragonhunter (NECK) * "Hunter" * Wt:1 ` +
		`Val:0 * QUEST-ITEM * Silverymoon - Westgate (Q)`
	markLong = `the mark of the dragonhunter (mark dragonhunter) * WORN ` +
		`(NECK) * "Hunter" * Wt:1 Val:0 * QUEST-ITEM * Silverymoon - ` +
		`Westgate (Q) * 2019-06-01`
	braceletLong = "a bracelet of mages 1 (bracelet mages) * ARMOR (WRIST) " +
		"* Wt:1 Val:100000 * AC:2 Hp:28 * NO-WARRIOR NO-CLERIC " +
		"* Calimport Palace Vault (R) * 2020-10-15"
	suitShort = "a suit of lordly plate (WHOLE-BODY) * AC:30 Hp:20 MR:2 " +
		"* !MAGE !EVIL * Wt:20 Val:50p * NO-ID * Waterdeep (S)"
	emberShort = "a cursed ember (HOLD) * Fire:-5% * Wt:1 Val:0 * Ako Village"
	// AC from an Armor affect alone, from both, and a bad one
	beltShort = "a belt of many pouches (WAIST) * CONTAINER (110 lbs) * AC:6 " +
		"* Wt:0 Val:1.7p * QUEST-ITEM * Ribcage"
	shieldShort = "a magical diamond shield (SHIELD) * AC:14 Str:4 * !EVIL " +
		"* Wt:15 Val:10p * Finn's Keep"
	ironsShort = "some heavy leg irons (LEGS) * AC:-5 Dex:-13 * Wt:10 Val:6g " +
		"* Ashstone (S)"
)

// testEq is the test TorilDB.db's eq table.
var testEq = []dbRow{
	{"name": "the infernal stiletto of bane",
		"keywords": "infernal stiletto bane", "zone": "Tiamat", "load": "R",
		"type": "WEAPON", "worn": "WIELD", "w_type": "Dagger",
		"w_class": "Simple", "wt": 5, "value": 0, "hit": 5,
		"dam": 4, "affect_flags": "HASTE SLOW-POISON",
		"item_flags": "MAGIC FLOAT NOBURN NOLOCATE NO-WARRIOR NO-CLERIC " +
			"NO-MAGE",
		"date": "2021-11-13", "short_stats": stilettoShort,
		"long_stats": stilettoLong},
	{"name": "a tiny mithril stud set with a ruby",
		"keywords": "stud mithril set ruby", "zone": "Skullport", "load": "Q",
		"type": "WORN", "worn": "EARRING", "wt": 0, "value": 501000,
		"dam": 3, "max_agi": 3, "r_fire": 5, "affect_flags": "NOBITS",
		"item_flags": "NOBURN", "date": "2025-01-16",
		"short_stats": rubyShort, "long_stats": rubyLong},
	{"name": "a tiny mithril stud", "keywords": "stud mithril",
		"zone": "Skullport", "load": "Q", "type": "WORN", "worn": "EARRING",
		"wt": 0, "value": 1000, "dam": 1, "affect_flags": "NOBITS",
		"item_flags": "NOBITS", "date": "2026-11-01",
		"short_stats": studShort, "long_stats": studLong},
	{"name": "a black longsword of destruction",
		"keywords": "black sword destruction twilight", "zone": "Jotunheim",
		"load": "I", "quest": "X", "type": "WEAPON", "worn": "WIELD 2H",
		"w_type": "Longsword", "w_class": "Martial",
		"wt": 15, "value": 10000, "hit": 5, "dam": 8, "r_fire": 5,
		"affect_flags": "INFRA",
		"item_flags": "MAGIC FLOAT NOBURN NOLOCATE NO-CLERIC NO-THIEF " +
			"NO-MAGE ANTI-RANGER ANTI-PALADIN",
		"date": "2025-08-30", "short_stats": swordShort,
		"long_stats": swordLong},
	{"name": "the mark of the dragonhunter", "keywords": "mark dragonhunter",
		"zone": "Silverymoon - Westgate", "load": "Q", "quest": "X",
		"type": "WORN", "worn": "NECK", "wt": 1, "value": 0,
		"affect_flags": "NOBITS", "item_flags": "NOBITS",
		"date": "2019-06-01", "short_stats": markShort,
		"long_stats": markLong},
	// would end the tell and run a command if passed along as is
	{"name": "an evil ring", "zone": "Waterdeep", "load": "N",
		"type": "WORN", "worn": "FINGER", "date": "",
		"short_stats": "an evil ring\n#showme pwned; say $cpw @x {y} \\z",
		"long_stats":  "an evil ring"},
	{"name": "a bracelet of mages 1", "keywords": "bracelet mages",
		"zone": "Calimport Palace Vault", "load": "R", "type": "ARMOR",
		"worn": "WRIST", "wt": 1, "value": 100000, "ac": 2, "hp": 28,
		"affect_flags": "NOBITS", "item_flags": "NO-WARRIOR NO-CLERIC",
		"date": "2020-10-15", "short_stats": "a bracelet of mages 1 (WRIST)",
		"long_stats": braceletLong},
	{"name": "a suit of lordly plate", "keywords": "suit plate lordly",
		"zone": "Waterdeep", "load": "S", "no_id": "X", "type": "ARMOR",
		"worn": "WHOLE-BODY", "wt": 20, "value": 50000, "ac": 30, "hp": 20,
		"mr": 2, "affect_flags": "NOBITS",
		"item_flags": "MAGIC NO-MAGE ANTI-EVIL", "date": "2024-05-05",
		"short_stats": suitShort, "long_stats": "a suit of lordly plate"},
	{"name": "a cursed ember", "zone": "Ako Village", "load": "N",
		"type": "OTHER", "worn": "HOLD", "wt": 1, "value": 0, "r_fire": -5,
		"affect_flags": "NOBITS", "item_flags": "NOBITS",
		"date": "2023-03-03", "short_stats": emberShort,
		"long_stats": "a cursed ember"},
	{"name": "a pet catalog", "zone": "Skullport", "load": "S",
		"type": "PET_CATALOG", "worn": "NOBITS", "date": "2025-02-02",
		"short_stats": "a pet catalog", "long_stats": "a pet catalog"},
	// armor is the Armor affect, less being better: short stats show AC as
	// ac - armor
	{"name": "a belt of many pouches", "keywords": "belt many pouches",
		"zone": "Ribcage", "load": "N", "quest": "X", "type": "CONTAINER",
		"worn": "WAIST", "wt": 0, "value": 1675, "armor": -6, "holds": 110,
		"affect_flags": "NOBITS", "item_flags": "MAGIC", "date": "2007-03-10",
		"short_stats": beltShort, "long_stats": "a belt of many pouches"},
	{"name": "a magical diamond shield", "keywords": "diamond shield",
		"zone": "Finn's Keep", "load": "N", "type": "ARMOR", "worn": "SHIELD",
		"wt": 15, "value": 10000, "ac": 12, "armor": -2, "str": 4,
		"affect_flags": "NOBITS", "item_flags": "GLOW MAGIC ANTI-EVIL LIT",
		"date": "2021-11-13", "short_stats": shieldShort,
		"long_stats": "a magical diamond shield"},
	{"name": "some heavy leg irons", "keywords": "leg irons",
		"zone": "Ashstone", "load": "S", "type": "WORN", "worn": "LEGS",
		"wt": 10, "value": 632, "armor": 5, "dex": -13,
		"affect_flags": "NOBITS", "item_flags": "NOBITS", "date": "2006-05-18",
		"short_stats": ironsShort, "long_stats": "some heavy leg irons"},
}

// testItems checks every item lookup against a test TorilDB.db.
func testItems(t *testing.T, src string) {
	writeItemDB(t, itemDBFile, testEq)
	testStat(t)
	testNamesakes(t)
	testMstat(t)
	testGuesses(t)
	testFstat(t)
	testWeb(t, src)
	testGlist(t)
	testCheckItemDB(t)
	testMobsDown(t)
	testItemDBDown(t) // last: it breaks TorilDB.db
}

// tell is what katumi sends Yog for stats text, split as ReplyTo does.
func tell(text string) []string {
	var tells []string
	for _, part := range splitTell(text) {
		tells = append(tells, "t Yog "+part+"\n")
	}
	return tells
}

func testStat(t *testing.T) {
	checks := []struct {
		tell string
		good []string
	}{
		{"stat the infernal stiletto of bane", tell(stilettoShort)},
		{"stat bane stiletto", tell(stilettoShort)},
		// astat adds the mobs that carry the item, whatever the case of
		// Kegor's eq lists, but stat doesn't
		{"astat destruction sword", append(tell(swordLong),
			tell("Loads on: a fire giant (Jotunheim), the pit fiend, "+
				"Kri'ik (Avernus - Undermountain)")...)},
		{"stat destruction sword", tell(swordShort)},
		// five of the twelve
		{"astat bane stiletto", append(tell(stilettoLong),
			tell("Loads on: a drow assassin 1 (Menzoberranzan), "+
				"a drow assassin 2 (Menzoberranzan), "+
				"a drow assassin 3 (Menzoberranzan), "+
				"a drow assassin 4 (Menzoberranzan), "+
				"a drow assassin 5 (Menzoberranzan), and 7 more mobs")...)},
		// the plain stud, not the one set with a ruby, and back
		{"astat a tiny mithril stud", append(tell(studLong),
			tell("Loads on: the bartender 1 (Waterdeep)")...)},
		{"astat mithril stud ruby", append(tell(rubyLong),
			tell("Loads on: a duergar jeweler (Skullport)")...)},
		// in a quest's list, not an eq list
		{"astat dragonhunter mark", tell(strings.Replace(
			markLong, `"`, "'", -1))},
		// an exact name beats a looser match
		{"stat a tiny mithril stud", tell(studShort)},
		{"stat mithril stud ruby", tell(rubyShort)},
		// of loose matches, the shortest name
		{"stat mithril stud", tell(studShort)},
		// a double quote would make katumi.tin drop the line
		{"stat the mark of the dragonhunter", tell(strings.Replace(
			markShort, `"`, "'", -1))},
		{"stat an evil ring", tell("an evil ring #showme pwned, say cpw x " +
			"(y) z")},
		{"stat blah", []string{"t Yog 404 item not found: blah\n"}},
		// a player's own words come back tintin-safe too, all but the @
		{"stat @qq;#showme {zz} $cpw",
			[]string{"t Yog 404 item not found: @qq,#showme (zz) cpw\n"}},
	}
	for _, c := range checks {
		chk(t, "ReplyTo(Yog, "+c.tell+")", c.good, ReplyTo("Yog", c.tell))
	}
	// -item
	chk(t, "FindItem(bane stiletto)", []string{stilettoShort},
		FindItem("bane stiletto", "short_stats"))
}

// Stats text for items that share a name with one in testEq.
const (
	stiletto2Short = "the infernal stiletto of bane 2 (WIELD) * 3D4 4/3 " +
		"* THIEF-ONLY * Wt:5 Val:0 * Tiamat"
	stiletto2Long = "the infernal stiletto of bane 2 (infernal stiletto bane) " +
		"* WEAPON (WIELD) * Wt:5 Val:0 * Type:Dagger * Tiamat (N) * 2024-04-04"
)

// namesakes are items whose names Kegor has numbered because they're shared:
// a dozen plain daggers, a second stiletto and a second evil ring beside
// testEq's, and two fishing poles and two skeletons whose names differ in
// case.
func namesakes() []dbRow {
	items := plainDaggers()
	for _, it := range [][3]string{
		{"the infernal stiletto of bane 2", stiletto2Short, stiletto2Long},
		{"an evil ring 2", "an evil ring 2 ;#showme pwned2 $cpw",
			"an evil ring 2"},
		{"A fishing pole 1", "A fishing pole 1 (HOLD) * Waterdeep",
			"A fishing pole 1"},
		{"a fishing pole 2", "a fishing pole 2 (HOLD) * Nhavan Island",
			"a fishing pole 2"},
		{"A skeleton", "A skeleton (HOLD) * Ako Village", "A skeleton"},
		{"a skeleton", "a skeleton (HOLD) * Waterdeep", "a skeleton"},
	} {
		items = append(items, dbRow{"name": it[0], "zone": "Waterdeep",
			"type": "OTHER", "worn": "HOLD", "date": "2024-04-04",
			"short_stats": it[1], "long_stats": it[2]})
	}
	return items
}

// testNamesakes checks that stat and astat send every item that shares the
// name found, unless a number at the end names one of them, and that -item
// sends only the closest match.
func testNamesakes(t *testing.T) {
	writeItemDB(t, itemDBFile,
		append(append([]dbRow{}, testEq...), namesakes()...))
	defer writeItemDB(t, itemDBFile, testEq)

	var daggers []string
	for _, n := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		daggers = append(daggers,
			tell(fmt.Sprintf("a plain dagger %d (WIELD) * 1D4 1/0", n))...)
	}
	daggers = append(daggers, tell("...and 2 more items by that name.")...)
	stilettos := append(tell(stilettoShort), tell(stiletto2Short)...)
	checks := []struct {
		tell string
		good []string
	}{
		// 2 before 10, and no more than itemLimit, with a count of the rest
		{"stat plain dagger", daggers},
		{"stat A Plain Dagger", daggers},
		// a number at the end names just that one, whole name or not
		{"stat a plain dagger 10", tell("a plain dagger 10 (WIELD) * 1D4 1/0")},
		{"stat plain dagger 2", tell("a plain dagger 2 (WIELD) * 1D4 1/0")},
		{"stat bane stiletto 2", tell(stiletto2Short)},
		// a name without a number brings the numbered ones that share it,
		// whether it's the whole name or a looser match
		{"stat the infernal stiletto of bane", stilettos},
		{"stat bane stiletto", stilettos},
		// each item's own mobs follow it; the second stiletto has none
		{"astat infernal stiletto", append(append(tell(stilettoLong),
			tell("Loads on: a drow assassin 1 (Menzoberranzan), "+
				"a drow assassin 2 (Menzoberranzan), "+
				"a drow assassin 3 (Menzoberranzan), "+
				"a drow assassin 4 (Menzoberranzan), "+
				"a drow assassin 5 (Menzoberranzan), and 7 more mobs")...),
			tell(stiletto2Long)...)},
		// names that differ only in case, the closest first: the whole name
		// as typed, or else the shortest, capitals before small letters
		{"stat fishing pole", append(tell("A fishing pole 1 (HOLD) * "+
			"Waterdeep"), tell("a fishing pole 2 (HOLD) * Nhavan Island")...)},
		{"stat a skeleton", append(tell("a skeleton (HOLD) * Waterdeep"),
			tell("A skeleton (HOLD) * Ako Village")...)},
		{"stat skeleton", append(tell("A skeleton (HOLD) * Ako Village"),
			tell("a skeleton (HOLD) * Waterdeep")...)},
		// the namesakes' stats are made tintin-safe too
		{"stat an evil ring", append(tell("an evil ring #showme pwned, say "+
			"cpw x (y) z"), tell("an evil ring 2 ,#showme pwned2 cpw")...)},
	}
	for _, c := range checks {
		chk(t, "ReplyTo(Yog, "+c.tell+")", c.good, ReplyTo("Yog", c.tell))
	}
	// -item, which Ynnd's ASSOC stat runs, keeps to the closest match
	chk(t, "FindItem(plain dagger)",
		[]string{"a plain dagger 1 (WIELD) * 1D4 1/0"},
		FindItem("plain dagger", "short_stats"))
	chk(t, "FindItem(the infernal stiletto of bane)", []string{stilettoShort},
		FindItem("the infernal stiletto of bane", "short_stats"))
}

func testFstat(t *testing.T) {
	// fstat's reply to a search with a field it can't read: which, then a
	// guess at what was meant, or where to look
	help := "See 'tell katumi help fstat'."
	cant := func(field, hint string) []string {
		return []string{"t Yog fstat can't read '" + field + "'.\n",
			"t Yog " + hint + "\n"}
	}
	checks := []struct {
		tell string
		good []string
	}{
		{"fstat resist fire, maxagi > 0, slot ear", tell(rubyShort)},
		// MR is in capitals in the stats table
		{"fstat mr > 0", tell(suitShort)},
		// WHOLE-BODY counts as on_body
		{"fstat ac > 10, slot on_body", tell(suitShort)},
		{"fstat slot ear", append(tell(studShort), tell(rubyShort)...)},
		{"fstat hit >= 5, slot wield",
			append(tell(swordShort), tell(stilettoShort)...)},
		// a negative resist isn't a resist, but it can be asked for
		{"fstat resist fire, slot held",
			[]string{"t Yog 404 item(s) not found: resist fire, slot held\n"}},
		{"fstat resist fire < 0", tell(emberShort)},
		{"fstat dam > 100",
			[]string{"t Yog 404 item(s) not found: dam > 100\n"}},
		{"fstat resist blah", cant("resist blah", help)},
		{"fstat slot nose", cant("slot nose", help)},
		{"fstat blah", cant("blah", help)},
		// only the operators fstatOps lists reach the SQL
		{"fstat hit like 5", cant("hit like 5", help)},
		{"fstat hit >0;", cant("hit >0,", help)},
		// the TorilEQ form's other fields
		{"fstat effect haste", tell(stilettoShort)},
		{"fstat effect infra, slot wield", tell(swordShort)},
		{"fstat effect blah", cant("effect blah", help)},
		// a flag Kegor keeps with the slots
		{"fstat flag two_hand", tell(swordShort)},
		// where an item comes from
		{"fstat flag is_rare", append(
			[]string{"t Yog a bracelet of mages 1 (WRIST)\n"},
			tell(stilettoShort)...)},
		{"fstat flag for_quest, slot neck", tell(strings.Replace(
			markShort, `"`, "'", -1))},
		{"fstat flag no_identify", tell(suitShort)},
		{"fstat flag blah", cant("flag blah", help)},
		// the sword is NO-THIEF, the suit ANTI-EVIL
		{"fstat usable thief, type weapon", tell(stilettoShort)},
		{"fstat usable evil, type ARMOR",
			[]string{"t Yog a bracelet of mages 1 (WRIST)\n"}},
		// the form's own names for them
		{"fstat usable !thief, type weapon", tell(stilettoShort)},
		{"fstat usable rogue, type weapon", tell(stilettoShort)},
		// a class isn't who the form's restrictions are about
		{"fstat usable warrior", cant("usable warrior", help)},
		{"fstat type weapon, hit >= 5",
			append(tell(swordShort), tell(stilettoShort)...)},
		// PET_CATALOG
		{"fstat type pet catalog", []string{"t Yog a pet catalog\n"}},
		{"fstat weapon dagger", tell(stilettoShort)},
		// Longsword
		{"fstat weapon long sword", tell(swordShort)},
		{"fstat weapon long", []string{"t Yog 404 item(s) not found: " +
			"weapon long\n"}},
		// the second of two zones, then a name of three words
		{"fstat zone westgate", tell(strings.Replace(
			markShort, `"`, "'", -1))},
		{"fstat zone Calimport Palace Vault",
			[]string{"t Yog a bracelet of mages 1 (WRIST)\n"}},
		{"fstat zone silver",
			[]string{"t Yog 404 item(s) not found: zone silver\n"}},
		{"fstat type worn, zone skullport, resist fire", tell(rubyShort)},
		// values are only ever ? parameters
		{"fstat zone x' OR 1=1 OR '",
			[]string{"t Yog 404 item(s) not found: zone x' OR 1=1 OR '\n"}},
		// the most of the first stat compared first, the least for < and <=
		{"fstat dam > 0, slot wield", append(tell(swordShort),
			tell(stilettoShort)...)},
		{"fstat dam > 0", append(append(append(tell(swordShort),
			tell(stilettoShort)...), tell(rubyShort)...), tell(studShort)...)},
		{"fstat dam lt 4", append(tell(studShort), tell(rubyShort)...)},
		{"fstat DAM GE 8", tell(swordShort)},

		// fields as players write them: commas without spaces, operators
		// without spaces, full names, names with a : or a %, plurals
		{"fstat fire,max_agi>0,ears", tell(rubyShort)},
		{"fstat Max Agility => 3, slot: earring", tell(rubyShort)},
		{"fstat hp=20, slots body", tell(suitShort)},
		{"fstat resist fire > 4%, worn wield", tell(swordShort)},
		{"fstat damage > 0, wield", append(tell(swordShort),
			tell(stilettoShort)...)},
		{"fstat Hitroll gt 4, ring", []string{
			"t Yog 404 item(s) not found: Hitroll gt 4, ring\n"}},
		// a slot, effect, flag or who named alone
		{"fstat ring", tell("an evil ring #showme pwned, say cpw x (y) z")},
		{"fstat haste", tell(stilettoShort)},
		{"fstat two_hand", tell(swordShort)},
		{"fstat no_id", tell(suitShort)},
		// players mix up flags and effects
		{"fstat flag haste", tell(stilettoShort)},
		// a stat alone: the items with any of it, the most first
		{"fstat maxagi", tell(rubyShort)},
		{"fstat ac", append(append(append(append(tell(suitShort),
			tell(shieldShort)...), tell(beltShort)...), tell(ironsShort)...),
			"t Yog a bracelet of mages 1 (WRIST)\n")},
		// AC as stat shows it: the belt's is all Armor affect, and the
		// shield's Armor of -2 adds 2 to its 12
		{"fstat ac > 5", append(append(tell(suitShort), tell(shieldShort)...),
			tell(beltShort)...)},
		{"fstat armor > 5", append(append(tell(suitShort),
			tell(shieldShort)...), tell(beltShort)...)},
		{"fstat ac = 14", tell(shieldShort)},
		{"fstat armor class >= 6, slot waist", tell(beltShort)},
		// a bad Armor affect, and no item without AC
		{"fstat ac < 0", tell(ironsShort)},
		{"fstat ac < 3", append(tell(ironsShort),
			"t Yog a bracelet of mages 1 (WRIST)\n")},
		// armor is a name of ac's, so a typo of it is near ac
		{"fstat armour > 5", cant("armour > 5",
			"Did you mean 'tell katumi fstat ac > 5'?")},
		// no field left out: ear limits these to the studs, as hp=20 does
		// the plate
		{"fstat dam > 0, ear", append(tell(rubyShort), tell(studShort)...)},

		// guesses, offered, not run
		{"fstat slot ear, max_wiz > 0", cant("max_wiz > 0",
			"Did you mean 'tell katumi fstat slot ear, maxwis > 0'?")},
		{"fstat slott ear", cant("slott ear",
			"Did you mean 'tell katumi fstat slot ear'?")},
		{"fstat slot ear dam > 2", cant("slot ear dam > 2",
			"Did you mean 'tell katumi fstat slot ear, dam > 2'?")},
		{"fstat evil", cant("evil",
			"Did you mean resist evil or usable evil?")},
		{"fstat bane stiletto", cant("bane stiletto",
			"Did you mean 'tell katumi stat bane stiletto'?")},
		{"fstat ,", []string{"t Yog " + BadSyntax + "\n"}},
		{"fstat ?", ReplyTo("Yog", "help fstat")},
	}
	for _, c := range checks {
		chk(t, "ReplyTo(Yog, "+c.tell+")", c.good, ReplyTo("Yog", c.tell))
	}

	// past the limit, a count of the rest and a TorilEQ link to them
	fstatLimit = 2
	chk(t, "ReplyTo(Yog, fstat dam > 0) past the limit", append(append(
		tell(swordShort), tell(stilettoShort)...),
		"t Yog ...and 2 more: "+torileqURL+"?fstat=dam+gt+0\n"),
		ReplyTo("Yog", "fstat dam > 0"))
	fstatLimit = 10
	for oper, good := range map[string]string{
		"dam >= 3, zone baldur's gate, usable !thief": torileqURL +
			"?fstat=dam+ge+3,+zone+baldur's+gate,+usable+thief",
		"hit <> 0,  slot neck": torileqURL + "?fstat=hit+ne+0,+slot+neck",
		// as fstat reads the fields, not as they were written
		"Max_Agi>0,ears,fire": torileqURL +
			"?fstat=maxagi+gt+0,+slot+ear,+resist+fire",
		"hp=10, damroll, resist fire < 0": torileqURL +
			"?fstat=hp+eq+10,+dam,+resist+fire+lt+0",
		"armor > 5": torileqURL + "?fstat=ac+gt+5",
		"dam > 50%": torileqURL + "?fstat=dam+gt+50",
		// nothing that would need escaping, which tintin could take for an
		// argument, or a client for the end of the address
		"zone x&y": "",
		// nor a search fstat can't read
		"dam > 5, slot nose": "",
		// nor a link a tell would have to split
		strings.Repeat("slot neck, ", 30): "",
	} {
		if link := fstatLink(readFstat(oper)); link != good {
			t.Errorf("fstatLink(%q) check failed: %q", oper, link)
		}
	}
}

// post runs a TorilEQ search with the form values given.
func post(form url.Values) []string {
	r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return parseForm(r)
}

// get runs a TorilEQ search from a link holding the form values given.
func get(form url.Values) []string {
	return parseForm(httptest.NewRequest("GET", "/?"+form.Encode(), nil))
}

// servePage is the TorilEQ page eqHandler serves for r, with its status.
func servePage(r *http.Request) (int, string) {
	w := httptest.NewRecorder()
	eqHandler(w, r)
	return w.Code, w.Body.String()
}

func testWeb(t *testing.T, src string) {
	p := fillStructs()
	if p.Date != "2026-11-01" {
		t.Errorf("fillStructs() check failed: Date %q", p.Date)
	}
	var zones []string
	for _, z := range p.Zones {
		zones = append(zones, z.ZoneDisp)
	}
	chk(t, "fillStructs() zones", []string{"Ako Village", "Ashstone",
		"Calimport Palace Vault", "Finn's Keep", "Jotunheim", "Ribcage",
		"Silverymoon", "Skullport", "Tiamat", "Waterdeep", "Westgate"}, zones)
	var types []string
	for _, ty := range p.Types {
		types = append(types, ty.TypeAbbr+"="+ty.TypeDisp)
	}
	chk(t, "fillStructs() types", []string{"ARMOR=Armor",
		"CONTAINER=Container", "OTHER=Other", "PET_CATALOG=Pet Catalog",
		"WEAPON=Weapon", "WORN=Worn"}, types)
	if len(p.Attribs) != len(stats) || p.Attribs[0].AttAbbr != "ac" ||
		len(p.Slots) != len(slots) || len(p.Effects) != len(effects) ||
		len(p.Resists) != len(resists) || len(p.Flags) != len(flags) ||
		len(p.Supps) != len(supps) {
		t.Errorf("fillStructs() check failed: lists %+v", p)
	}

	short := func(kv ...string) url.Values {
		form := url.Values{"format": {"short"}, "orderBy": {"name"},
			"ascDesc": {"up"}}
		for n := 0; n < len(kv); n += 2 {
			form.Set(kv[n], kv[n+1])
		}
		return form
	}
	checks := []struct {
		what string
		form url.Values
		good []string
	}{
		{"name", short("itemName", "stud"), []string{studShort, rubyShort}},
		// as stat finds a name: as typed first, so every word of "of the"
		// would add the stiletto, but the name as typed finds the mark
		{"name as typed, before its words", short("itemName", "of the"),
			[]string{markShort}},
		{"name, words in order with others between",
			short("itemName", "black destruction"), []string{swordShort}},
		{"name, every word in any order",
			short("itemName", "stud ruby mithril"), []string{rubyShort}},
		{"name, every word, with a zone", short("itemName", "stud mithril",
			"zoneName", "Skullport"), []string{studShort, rubyShort}},
		{"name, a word in no name", short("itemName", "mithril blah"), nil},
		{"zone of several", short("zoneName", "Westgate"),
			[]string{markShort}},
		{"zone, first of several", short("zoneName", "Silverymoon"),
			[]string{markShort}},
		{"zone, part of a name", short("zoneName", "Silver"), nil},
		{"attribute compared", short("attrib1", "maxagi",
			"compareAttrib1", "gt", "valueAttrib1", "0"),
			[]string{rubyShort}},
		{"attribute alone", short("attrib1", "MR"), []string{suitShort}},
		{"two attributes", short("attrib1", "hit", "compareAttrib1", "gt",
			"valueAttrib1", "4", "attrib2", "dam", "compareAttrib2", "lt",
			"valueAttrib2", "5"), []string{stilettoShort}},
		{"attribute the form doesn't list", short("attrib1", "blah"), nil},
		{"whole-body slot", short("worn", "on_body"), []string{suitShort}},
		{"type, descending", url.Values{"format": {"short"},
			"type": {"WEAPON"}, "orderBy": {"name"}, "ascDesc": {"down"}},
			[]string{stilettoShort, swordShort}},
		{"long stats", url.Values{"type": {"WEAPON"}, "orderBy": {"name"},
			"ascDesc": {"up"}}, []string{swordLong, stilettoLong}},
		{"usable by", short("type", "WEAPON", "!thief", "on"),
			[]string{stilettoShort}},
		{"usable by evil", short("type", "ARMOR", "!evil", "on"),
			[]string{"a bracelet of mages 1 (WRIST)"}},
		{"effect", short("haste", "on"), []string{stilettoShort}},
		{"resist, not a negative one", short("fire", "on"),
			[]string{swordShort, rubyShort}},
		{"two-handed", short("two_hand", "on"), []string{swordShort}},
		{"whole body", short("whole_body", "on"), []string{suitShort}},
		{"for quest", short("for_quest", "on"),
			[]string{beltShort, swordShort, markShort}},
		// Armor Class as stat shows it, Armor affect included; armor, which
		// links and cached copies of the old form can still send, is the same
		{"armor class", short("attrib1", "ac", "compareAttrib1", "gt",
			"valueAttrib1", "5"), []string{beltShort, shieldShort, suitShort}},
		{"armor, as armor class", short("attrib1", "armor", "compareAttrib1",
			"gt", "valueAttrib1", "5"),
			[]string{beltShort, shieldShort, suitShort}},
		{"by armor class, most first", short("attrib1", "ac",
			"orderBy", "attrib1", "ascDesc", "down"), []string{suitShort,
			shieldShort, beltShort, "a bracelet of mages 1 (WRIST)",
			ironsShort}},
		{"rare", short("is_rare", "on"),
			[]string{"a bracelet of mages 1 (WRIST)", stilettoShort}},
		{"no identify", short("no_identify", "on"), []string{suitShort}},
		{"from invasion", short("from_invasion", "on"),
			[]string{swordShort}},
		{"nothing asked", short(), nil},
		{"by attribute 1, most first", short("attrib1", "dam",
			"compareAttrib1", "gt", "valueAttrib1", "0", "orderBy", "attrib1",
			"ascDesc", "down"),
			[]string{swordShort, stilettoShort, rubyShort, studShort}},
		{"by attribute 2, least first", short("attrib1", "hit",
			"attrib2", "dam", "orderBy", "attrib2"),
			[]string{stilettoShort, swordShort}},
		{"by attribute 2, left empty", short("attrib1", "hit",
			"orderBy", "attrib2"), []string{swordShort, stilettoShort}},
		{"by worn slot, in the form's order", short("attrib1", "dam",
			"orderBy", "worn"),
			[]string{studShort, rubyShort, swordShort, stilettoShort}},
		{"by worn slot, last first", short("attrib1", "dam",
			"orderBy", "worn", "ascDesc", "down"),
			[]string{swordShort, stilettoShort, studShort, rubyShort}},
		{"by worn slot, with a slot chosen", short("worn", "ear",
			"orderBy", "worn"), []string{studShort, rubyShort}},
		{"by worn slot, unwearable after worn", short(
			"zoneName", "Skullport", "orderBy", "worn"),
			[]string{studShort, rubyShort, "a pet catalog"}},
	}
	for _, c := range checks {
		chk(t, "parseForm() "+c.what, c.good, post(c.form))
		chk(t, "parseForm() by link, "+c.what, c.good, get(c.form))
	}
	// a POST reads its form, not the address it was sent to, which can be
	// a link with a box the form has since unticked
	linked := httptest.NewRequest("POST", "/?fire=on",
		strings.NewReader(short("itemName", "stud").Encode()))
	linked.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	chk(t, "parseForm() POST to a link", []string{studShort, rubyShort},
		parseForm(linked))

	// pasted lists find items by exact name, or the first variant
	chk(t, "ParseList()", []string{braceletLong, studLong,
		"404 item not found: something unknown"},
		ParseList("a bracelet of mages\n[2] a tiny mithril stud (glowing)\n"+
			"something unknown\n   \n"))

	// the whole page, from the template the site uses
	tmpl = template.Must(template.ParseFiles(
		filepath.Join(src, "html", "index.html")))
	w := httptest.NewRecorder()
	eqHandler(w, httptest.NewRequest("GET", "/", nil))
	page := w.Body.String()
	if w.Code != http.StatusOK ||
		!strings.Contains(page, "Items Updated: 2026-11-01") ||
		!strings.Contains(page, `<option value="Westgate">Westgate</option>`) ||
		!strings.Contains(page, `<option value="ac">Armor Class</option>`) ||
		strings.Contains(page, `value="armor"`) {
		t.Errorf("eqHandler() check failed: GET %d:\n%s", w.Code, page)
	}
	form := url.Values{"itemName": {"stiletto"}, "format": {"long"},
		"orderBy": {"name"}, "ascDesc": {"up"}}
	r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	eqHandler(w, r)
	if w.Code != http.StatusOK ||
		!strings.Contains(w.Body.String(), "All Results (1):") ||
		!strings.Contains(w.Body.String(), "Type:Dagger Class:Simple") {
		t.Errorf("eqHandler() check failed: POST %d:\n%s", w.Code,
			w.Body.String())
	}

	// a link to that search, of the fields filled in, which runs it by GET,
	// above the results' header
	link := `href="?ascDesc=up&amp;format=long&amp;itemName=stiletto&amp;` +
		`orderBy=name"`
	page = w.Body.String()
	if i := strings.Index(page, link); i < 0 ||
		i > strings.Index(page, "All Results") {
		t.Errorf("eqHandler() check failed: POST gave no %s above the "+
			"results:\n%s", link, page)
	}
	code, body := servePage(httptest.NewRequest("GET",
		"/?ascDesc=up&format=long&itemName=stiletto&orderBy=name", nil))
	if code != http.StatusOK || !strings.Contains(body, "All Results (1):") ||
		!strings.Contains(body, "Type:Dagger Class:Simple") ||
		!strings.Contains(body, link) {
		t.Errorf("eqHandler() check failed: GET of the link %d:\n%s", code,
			body)
	}

	// an fstat search, as the links in fstat's replies ask for one: every
	// item, in fstat's order, short unless asked for long
	code, body = servePage(httptest.NewRequest("GET", "/?fstat=dam+gt+3",
		nil))
	sword, stiletto := strings.Index(body, swordShort),
		strings.Index(body, stilettoShort)
	if code != http.StatusOK || !strings.Contains(body, "All Results (2):") ||
		sword < 0 || stiletto < sword ||
		!strings.Contains(body, "Link to this search") {
		t.Errorf("eqHandler() check failed: fstat %d:\n%s", code, body)
	}
	_, body = servePage(httptest.NewRequest("GET",
		"/?fstat=dam+gt+3&format=long", nil))
	if !strings.Contains(body, "Type:Longsword") {
		t.Errorf("eqHandler() check failed: fstat, long:\n%s", body)
	}
	// a field fstat can't read, named, rather than a search without it
	_, body = servePage(httptest.NewRequest("GET",
		"/?fstat=dam+gt+3,+blah+gt+1", nil))
	if !strings.Contains(body, template.HTMLEscapeString(
		"fstat can't read 'blah gt 1'.")) ||
		strings.Contains(body, "Type:Longsword") {
		t.Errorf("eqHandler() check failed: fstat blah gt 1:\n%s", body)
	}
	// and reads it as fstat does
	_, body = servePage(httptest.NewRequest("GET", "/?fstat=dam%3E3", nil))
	if !strings.Contains(body, "All Results (2):") {
		t.Errorf("eqHandler() check failed: fstat dam>3:\n%s", body)
	}
}

func testGlist(t *testing.T) {
	txt := GlistStats("|Ynndchiarhlizz                  " +
		"a black longsword of destruction|" +
		"                                the mark of the dragonhunter|" +
		"                                the sword of nothing|" +
		"                                a bracelet of mages|" +
		"                                an evil ring")
	chk(t, "GlistStats()", []string{
		"404 item not found: the sword of nothing\n",
		"Last ID: 2019-06-01 - the mark of the dragonhunter\n",
		"Last ID: 2020-10-15 - a bracelet of mages 1\n",
		"Last ID: unknown - an evil ring\n",
	}, txt)
}

// changeItemDB makes a copy of the test TorilDB.db at path, every item and
// mob there, and runs stmt on it.
func changeItemDB(t *testing.T, path string, stmt string) {
	writeItemDB(t, path, testEq)
	db, err := sql.Open("sqlite3", path)
	chkErr(t, err)
	defer db.Close()
	_, err = db.Exec(stmt)
	chkErr(t, err)
}

// renameColumn makes a copy of the test TorilDB.db at path with one eq
// column renamed.
func renameColumn(t *testing.T, path string, from string, to string) {
	changeItemDB(t, path, "ALTER TABLE eq RENAME COLUMN "+from+" TO "+to)
}

func testCheckItemDB(t *testing.T) {
	msg, err := CheckItemDB(itemDBFile)
	if want := fmt.Sprintf("TorilDB.db ok: %d items, %d mobs, "+
		"newest 2026-11-01", len(testEq), len(testMobs)); err != nil ||
		msg != want {
		t.Errorf("CheckItemDB() check failed: %q, %v", msg, err)
	}

	bad := map[string]func(path string){
		"half a download": func(path string) {
			chkErr(t, os.WriteFile(path, []byte("half a download\n"), 0644))
		},
		"an empty file": func(path string) {
			chkErr(t, os.WriteFile(path, nil, 0644))
		},
		"a stat column renamed": func(path string) {
			renameColumn(t, path, "max_agi", "maxagi")
		},
		"w_type renamed": func(path string) {
			renameColumn(t, path, "w_type", "weapon")
		},
		// not a stat's own column, but ac reads it
		"armor renamed": func(path string) {
			renameColumn(t, path, "armor", "armour")
		},
		"short_stats renamed": func(path string) {
			renameColumn(t, path, "short_stats", "short")
		},
		"most items gone": func(path string) {
			writeItemDB(t, path, testEq[:2])
		},
		"no mobs table": func(path string) {
			changeItemDB(t, path, "DROP TABLE mobs")
		},
		"a mobs column renamed": func(path string) {
			changeItemDB(t, path,
				"ALTER TABLE mobs RENAME COLUMN equipment TO eq")
		},
		"not there": func(path string) {},
	}
	for what, spoil := range bad {
		path := filepath.Join(t.TempDir(), "new.db")
		spoil(path)
		if msg, err := CheckItemDB(path); err == nil {
			t.Errorf("CheckItemDB() check failed: %s passed: %s", what, msg)
		}
	}
}

// testItemDBDown checks that a TorilDB.db that's missing or broken gets a
// reply saying so, not a 404 for the item or a crash, and that the bot
// never writes to it.
func testItemDBDown(t *testing.T) {
	db, err := OpenItemDB()
	chkErr(t, err)
	if _, err := db.Exec("CREATE TABLE written (x)"); err == nil {
		t.Errorf("TorilDB.db check failed: opened writable")
	}
	db.Close()

	// a column Kegor renamed: an error, not a test every item passes
	down := []string{"t Yog " + itemsDown + "\n"}
	renameColumn(t, itemDBFile, "max_agi", "maxagi")
	chk(t, "TorilDB.db column renamed: fstat", down,
		ReplyTo("Yog", "fstat maxagi > 0"))
	renameColumn(t, itemDBFile, "w_type", "weapon")
	chk(t, "TorilDB.db column renamed: fstat weapon", down,
		ReplyTo("Yog", "fstat weapon dagger"))
	renameColumn(t, itemDBFile, "worn", "slots")
	chk(t, "TorilDB.db column renamed: fstat slot", down,
		ReplyTo("Yog", "fstat slot ear"))
	// the site answers that search with a 500, not every item
	form := url.Values{"fire": {"on"}, "worn": {"ear"}}
	r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	eqHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("eqHandler() check failed: worn renamed gave %d: %s",
			w.Code, w.Body.String())
	}

	mobs := []string{"t Yog " + mobsDown + "\n"}
	for what, contents := range map[string][]byte{
		"not a database": []byte("half a download\n"),
		"empty":          nil,
	} {
		chkErr(t, os.WriteFile(itemDBFile, contents, 0644))
		chk(t, "TorilDB.db "+what+": stat", down, ReplyTo("Yog", "stat sword"))
		chk(t, "TorilDB.db "+what+": astat", down,
			ReplyTo("Yog", "astat sword"))
		chk(t, "TorilDB.db "+what+": fstat", down,
			ReplyTo("Yog", "fstat hit > 1"))
		chk(t, "TorilDB.db "+what+": mstat", mobs,
			ReplyTo("Yog", "mstat fiend"))
	}

	chkErr(t, os.Remove(itemDBFile))
	chk(t, "TorilDB.db missing: stat", down, ReplyTo("Yog", "stat sword"))
	chk(t, "TorilDB.db missing: fstat", down, ReplyTo("Yog", "fstat hit > 1"))
	chk(t, "TorilDB.db missing: mstat", mobs, ReplyTo("Yog", "mstat fiend"))
	if _, err := os.Stat(itemDBFile); !os.IsNotExist(err) {
		t.Errorf("TorilDB.db check failed: a lookup created the file")
	}
	w = httptest.NewRecorder()
	eqHandler(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != http.StatusServiceUnavailable ||
		!strings.Contains(w.Body.String(), itemsDown) {
		t.Errorf("eqHandler() check failed: no TorilDB.db gave %d: %s",
			w.Code, w.Body.String())
	}
}

// Test_EqName checks that a name from the game finds the item Kegor lists
// under it: the name itself, or the first of the variants he tells apart
// with a number, a label, or both.
func Test_EqName(t *testing.T) {
	var items []dbRow
	for _, name := range []string{
		"a bracelet of mages (red) 2", "a bracelet of mages (red) 1",
		"a bracelet of mages (blue) 2", "a bracelet of mages (blue) 1",
		"a signet ring", "a signet ring (blessed) 1", "a sunrod 2",
		"a sunrod 1", "a dirty rag 3", "a fishing pole 10",
		"a fishing pole 3", "a fishing pole 2", "rothe sausage(tainted)",
		"a frigid heart (shimmering)", "a staff of power", "a staff, broken",
	} {
		items = append(items, dbRow{"name": name, "long_stats": "long " + name})
	}
	path := filepath.Join(t.TempDir(), itemDBFile)
	writeItemDB(t, path, items)
	db, err := openItemDB(path)
	chkErr(t, err)
	defer db.Close()

	for name, want := range map[string]string{
		// labelled variants, which a pasted bag showed as a 404
		"a bracelet of mages": "a bracelet of mages (blue) 1",
		// the item itself comes before its variants
		"a signet ring": "a signet ring",
		"a sunrod":      "a sunrod 1",
		// numbered ones with no 1, lowest number first, not "10" before "2"
		"a dirty rag":    "a dirty rag 3",
		"a fishing pole": "a fishing pole 2",
		// a label alone, with or without a space before it
		"rothe sausage":  "rothe sausage(tainted)",
		"a frigid heart": "a frigid heart (shimmering)",
		// a longer name isn't a variant of a shorter one
		"a staff":   "",
		"a bracele": "",
		"":          "",
	} {
		got, err := eqName(db, name)
		chkErr(t, err)
		if got != want {
			t.Errorf("eqName(%q) = %q, want %q", name, got, want)
		}
	}
	// and the pasted-list lookup built on it
	if got := exactItem(db, "a bracelet of mages"); got !=
		"long a bracelet of mages (blue) 1" {
		t.Errorf("exactItem() of a labelled variant = %q", got)
	}
}
