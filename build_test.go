package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// buildEq are items testBuild adds to testEq for the build designer: a
// whole-head helm, what it covers, a tail, a class's own weapon, a race's
// cloak, and items for each of the hands.
var buildEq = []dbRow{
	{"name": "a horned great helm", "zone": "Jotunheim", "load": "N",
		"type": "ARMOR", "worn": "WHOLE-HEAD", "wt": 8, "ac": 10, "hp": 5,
		"affect_flags": "NOBITS", "item_flags": "NO-MAGE",
		"date": "2024-01-01", "short_stats": "a horned great helm " +
			"(WHOLE-HEAD) * AC:10 Hp:5 * !MAGE * Wt:8 Val:0 * Jotunheim",
		"long_stats": "a horned great helm (horned helm) * ARMOR"},
	{"name": "a pair of goggles", "zone": "Gnome Village", "load": "S",
		"type": "WORN", "worn": "EYES", "wt": 1, "hp": 3,
		"affect_flags": "DET-INVIS", "item_flags": "ANTI-MALE",
		"date": "2024-01-01", "short_stats": "a pair of goggles (EYES) * " +
			"Hp:3 * DET-INVIS * Wt:1 Val:0 * Gnome Village (S)",
		"long_stats": "a pair of goggles"},
	{"name": "a dark blindfold", "zone": "Ashstone", "load": "N",
		"type": "WORN", "worn": "EYES", "wt": 0, "affect_flags": "BLINDNESS",
		"item_flags": "NOBITS", "date": "2024-01-01",
		"short_stats": "a dark blindfold (EYES) * BLINDNESS * Wt:0 Val:0",
		"long_stats":  "a dark blindfold"},
	{"name": "a jeweled tail ring", "zone": "Ashstone", "load": "N",
		"type": "WORN", "worn": "TAIL", "wt": 0, "hp": 10,
		"affect_flags": "NOBITS", "item_flags": "NOBITS",
		"date": "2024-01-01", "short_stats": "a jeweled tail ring (TAIL) * " +
			"Hp:10 * Wt:0 Val:0 * Ashstone", "long_stats": "a jeweled tail ring"},
	{"name": "a phantom blade", "zone": "Baldur's Gate", "load": "Q",
		"type": "WEAPON", "worn": "WIELD", "wt": 3, "dam": 2,
		"usable_by": "Wizard:Illusionist", "affect_flags": "NOBITS",
		"item_flags": "MAGIC NO-WARRIOR NO-CLERIC NO-THIEF", "date": "2026-06-26",
		"short_stats": "a phantom blade (WIELD) * 0/2 * Wizard:Illusionist * " +
			"Wt:3 Val:0 * Baldur's Gate (Q)",
		"long_stats": "a phantom blade"},
	{"name": "a wizard's inkwell", "zone": "Baldur's Gate", "load": "S",
		"type": "OTHER", "worn": "HOLD", "wt": 1, "usable_by": "Wizard",
		"affect_flags": "NOBITS", "item_flags": "NOBITS", "date": "2026-06-26",
		"short_stats": "a wizard's inkwell (HOLD) * Wizard * Wt:1 Val:0",
		"long_stats":  "a wizard's inkwell"},
	{"name": "an enchanter's sigil", "zone": "Baldur's Gate", "load": "S",
		"type": "OTHER", "worn": "HOLD", "wt": 1,
		"usable_by": "Wizard:Enchanter", "affect_flags": "NOBITS",
		"item_flags": "NOBITS", "date": "2026-06-26",
		"short_stats": "an enchanter's sigil (HOLD) * Wizard:Enchanter * " +
			"Wt:1 Val:0", "long_stats": "an enchanter's sigil"},
	{"name": "a troll-hide cloak", "zone": "Troll Hills", "load": "N",
		"type": "ARMOR", "worn": "ABOUT", "wt": 6, "hp": 15, "r_cold": 10,
		"affect_flags": "FLY", "item_flags": "ANTI-HUMAN ANTI-GOODRACE",
		"date": "2024-01-01", "short_stats": "a troll-hide cloak (ABOUT) * " +
			"Hp:15 * Cold:10% * FLY * Wt:6 Val:0 * Troll Hills",
		"long_stats": "a troll-hide cloak"},
	{"name": "a short sword", "zone": "Waterdeep", "load": "S",
		"type": "WEAPON", "worn": "WIELD", "wt": 4, "hit": 1, "dam": 1,
		"affect_flags": "NOBITS", "item_flags": "NO-THIEF",
		"date": "2024-01-01", "short_stats": "a short sword (WIELD) * 1D6 " +
			"1/1 * !THIEF * Wt:4 Val:1p * Waterdeep (S)",
		"long_stats": "a short sword"},
	{"name": "a gnarled staff", "zone": "Ashstone", "load": "N",
		"type": "STAFF", "worn": "HOLD 2H", "wt": 6, "mana": 20,
		"affect_flags": "NOBITS", "item_flags": "NOBITS",
		"date": "2024-01-01", "short_stats": "a gnarled staff (HOLD 2H) * " +
			"Mana:20 * Wt:6 Val:0 * Ashstone", "long_stats": "a gnarled staff"},
	{"name": "a yew longbow", "zone": "Ashstone", "load": "N",
		"type": "WEAPON", "worn": "WIELD 2H", "wt": 3,
		"affect_flags": "NOBITS", "item_flags": "NOBITS",
		"date": "2024-01-01", "short_stats": "a yew longbow (WIELD 2H) * " +
			"1D8 * Wt:3 Val:0 * Ashstone", "long_stats": "a yew longbow"},
}

// testBuild checks the build designer against a test TorilDB.db.
func testBuild(t *testing.T, src string) {
	writeItemDB(t, itemDBFile, append(append(append([]dbRow{}, testEq...),
		buildEq...), bestEq...))
	defer writeItemDB(t, itemDBFile, testEq)
	buildTmpl = template.Must(template.ParseFiles(
		filepath.Join(src, "html", "build.html")))

	db, err := OpenItemDB()
	chkErr(t, err)
	defer db.Close()
	testSlotItem(t, db)
	testBuildEval(t, db)
	testBuildChar(t, db)
	testBuildPaste(t, db)
	testBuildBest(t, db)
	testBuildPost(t)
	testBuildPage(t)
	testBuildNames(t)
}

func testSlotItem(t *testing.T, db *sql.DB) {
	checks := []struct{ what, name, slot, want string }{
		{"exact", "a tiny mithril stud", "ear1", "a tiny mithril stud"},
		{"anywhere in a name, the shortest", "stud", "ear2",
			"a tiny mithril stud"},
		{"every word in any order", "ruby stud", "ear1",
			"a tiny mithril stud set with a ruby"},
		{"in any case", "TINY MITHRIL STUD", "ear1", "a tiny mithril stud"},
		{"exact, worn elsewhere", "a tiny mithril stud", "neck1",
			"a tiny mithril stud"},
		{"loose, only what's worn there", "stud", "neck1", ""},
		{"a two-handed weapon as the primary", "longsword", "primary",
			"a black longsword of destruction"},
		{"not as the secondary", "longsword", "secondary", ""},
		{"whole body on the body", "lordly", "body", "a suit of lordly plate"},
		{"whole head on the head", "horned", "head", "a horned great helm"},
		{"two-handed, held", "staff", "held", "a gnarled staff"},
		{"not as the second held item", "staff", "held2", ""},
		{"one-handed, as the second held item", "ember", "held2",
			"a cursed ember"},
		{"nothing by that name", "a sword of nothing", "primary", ""},
	}
	for _, c := range checks {
		it, err := newFinder(db).slotItem(c.name,
			buildSlots[slotIndex(c.slot)])
		chkErr(t, err)
		got := ""
		if it != nil {
			got = it.name
		}
		if got != c.want {
			t.Errorf("slotItem() %s check failed: %q in %s gave %q, not %q",
				c.what, c.name, c.slot, got, c.want)
		}
	}
	it, err := itemNamed(db, "a tiny mithril stud")
	chkErr(t, err)
	if it == nil || it.short != "(EAR) * 0/1 * Wt:0 Val:1p * Skullport (Q)" ||
		it.vals["dam"] != 1 || len(it.vals) != 1 {
		t.Errorf("itemNamed() check failed: %+v", it)
	}

	// a request matches no more than maxLoose names loosely, though exact
	// names are always found, and each name once for each kind of slot
	f := newFinder(db)
	ear := buildSlots[slotIndex("ear1")]
	for n := 0; n < maxLoose; n++ {
		it, err := f.slotItem(fmt.Sprintf("nothing %d", n), ear)
		chkErr(t, err)
		if it != nil {
			t.Errorf("slotItem() check failed: nothing %d found %s", n, it.name)
		}
	}
	for name, want := range map[string]bool{"stud": false,
		"a tiny mithril stud": true, "nothing 1": false} {
		it, err := f.slotItem(name, ear)
		chkErr(t, err)
		if (it != nil) != want {
			t.Errorf("slotItem() past maxLoose check failed: %s: %+v", name, it)
		}
	}
	if f.loose != maxLoose {
		t.Errorf("slotItem() check failed: %d loose matches", f.loose)
	}
}

// warnings lists a set's warnings, a slot at a time: the slot's key, then
// each warning, with ? before a maybe.
func warnings(e setEval) []string {
	var ws []string
	for n, slotWarns := range e.warns {
		for _, w := range slotWarns {
			mark := ""
			if w.Maybe {
				mark = "?"
			}
			ws = append(ws, buildSlots[n].key+": "+mark+w.Text)
		}
	}
	return ws
}

// evalSets reads a build from link values, and evaluates its sets.
func evalSets(t *testing.T, db *sql.DB, v url.Values) []setEval {
	b := readBuild(v)
	f := newFinder(db)
	var evals []setEval
	for _, set := range b.sets {
		e, err := b.eval(f, set)
		chkErr(t, err)
		evals = append(evals, e)
	}
	return evals
}

// totals shows the totals of sets as text: each row's label, then each
// set's cell, its value with its differences, as in "Damroll: 8 | 12
// +4(better)".
func totals(evals []setEval) []string {
	var rows []string
	for _, row := range totalRows(evals) {
		var cells []string
		for _, c := range row.Cells {
			cell := c.Value
			for _, d := range c.Diffs {
				cell += " " + d.Text + "(" + d.Class + ")"
			}
			cells = append(cells, cell)
		}
		rows = append(rows, row.Label+": "+strings.Join(cells, " | "))
	}
	return rows
}

func testBuildEval(t *testing.T, db *sql.DB) {
	// three hands, and leg irons under a whole-body suit; the ruby's fire
	// resist and the ember's cancel out
	set := url.Values{"a.ear1": {"a tiny mithril stud"}, "a.ear2": {"ruby"},
		"a.body": {"lordly"}, "a.waist": {"belt"}, "a.shield": {"diamond"},
		"a.primary": {"stiletto"}, "a.held": {"ember"}, "a.legs": {"irons"},
		"a.finger1": {"an evil ring"}}
	evals := evalSets(t, db, set)
	hands := "The set needs 3 hands, and a character has two."
	chk(t, "eval() warnings", []string{
		"shield: " + hands, "primary: " + hands, "held: " + hands,
		"legs: The whole-body item worn on body covers this slot."},
		warnings(evals[0]))
	if n := evals[0].problems(); n != 4 {
		t.Errorf("problems() check failed: %d", n)
	}
	if len(evals) != 2 || evals[1].count != 0 {
		t.Errorf("eval() check failed: %d sets", len(evals))
	}
	chk(t, "totalRows() of one set", []string{"Items: 9 | ",
		"Armor Class: 45 | ", "Hitroll: 5 | ", "Damroll: 8 | ",
		"Hitpoints: 20 | ", "Max Agility: 3 | ", "Strength: 4 | ",
		"Dexterity: -13 | ", "Magic Resistance: 2 | ",
		"Effects: Haste, Slow Poison | ", "Equipment weight: 51 | "},
		totals(evals))

	// the same with a two-handed sword in place of the stiletto, ember and
	// shield: two hands, and how it differs from the first
	set.Set("b.ear1", "a tiny mithril stud")
	for _, k := range []string{"ear2", "body", "waist", "legs", "finger1"} {
		set.Set("b."+k, set.Get("a."+k))
	}
	set.Set("b.primary", "longsword")
	evals = evalSets(t, db, set)
	chk(t, "eval() warnings, two-handed", []string{
		"legs: The whole-body item worn on body covers this slot."},
		warnings(evals[1]))
	chk(t, "totalRows() of two sets", []string{"Items: 9 | 7 -2()",
		"Armor Class: 45 | 31 -14(worse)", "Hitroll: 5 | 5",
		"Damroll: 8 | 12 +4(better)", "Hitpoints: 20 | 20",
		"Max Agility: 3 | 3", "Strength: 4 | 0 -4(worse)",
		"Dexterity: -13 | -13", "Magic Resistance: 2 | 2",
		"Fire resist: 0% | 10% +10%(better)",
		"Effects: Haste, Slow Poison | Infravision +Infravision(better) " +
			"-Haste(worse) -Slow Poison(worse)",
		"Equipment weight: 51 | 45 -6(better)"}, totals(evals))

	// a whole-head helm covers the eyes, a two-handed staff takes both
	// hands, saves are better less, and blindness is no gain
	evals = evalSets(t, db, url.Values{"a.head": {"horned"},
		"a.eyes": {"goggles"}, "a.held": {"staff"}, "a.primary": {"short"},
		"b.eyes":      {"blindfold"},
		"b.secondary": {"a black longsword of destruction"},
		"b.finger1":   {"a pair of goggles"}})
	chk(t, "eval() warnings, whole head", []string{
		"eyes: The whole-head item worn on head covers this slot.",
		"primary: The set needs 3 hands, and a character has two.",
		"held: The set needs 3 hands, and a character has two."},
		warnings(evals[0]))
	chk(t, "eval() warnings, worn elsewhere", []string{
		"finger1: TorilEQ lists it as worn EYES, not here.",
		"secondary: It takes both hands, so it can only be the primary " +
			"weapon."}, warnings(evals[1]))
	effects := totals(evals)
	if want := "Effects: Detect Invisibility | Detect Invisibility, " +
		"Infravision, Blindness +Infravision(better) +Blindness(worse)"; !hasWord(effects, want) {
		t.Errorf("totalRows() check failed: no %q in %q", want, effects)
	}

	// two held items take both hands, a weapon with them makes three, and
	// an item that takes both hands can only be the first held item
	evals = evalSets(t, db, url.Values{"a.held": {"ember"},
		"a.held2": {"ember"}, "b.held": {"ember"}, "b.held2": {"ember"},
		"b.primary": {"short"}, "c.held2": {"a gnarled staff"}})
	chk(t, "eval() warnings, two held items", nil, warnings(evals[0]))
	chk(t, "eval() warnings, two held items and a weapon", []string{
		"primary: The set needs 3 hands, and a character has two.",
		"held: The set needs 3 hands, and a character has two.",
		"held2: The set needs 3 hands, and a character has two."},
		warnings(evals[1]))
	chk(t, "eval() warnings, two-handed as the second held item", []string{
		"held2: It takes both hands, so it can only be the first held item."},
		warnings(evals[2]))
	if evals[0].count != 2 || evals[0].vals["r:fire"] != -10 {
		t.Errorf("eval() two held items check failed: %d items, %d fire",
			evals[0].count, evals[0].vals["r:fire"])
	}

	// names that find nothing, and a set left empty between two
	evals = evalSets(t, db, url.Values{"a.head": {"a hat of nothing"},
		"c.tail": {"tail ring"}})
	chk(t, "eval() warnings, not found",
		[]string{"head: No item by that name fits this slot."},
		warnings(evals[0]))
	chk(t, "totalRows() with empty sets",
		[]string{"Items:  |  | 1", "Hitpoints:  |  | 10"}, totals(evals))
}

func testBuildChar(t *testing.T, db *sql.DB) {
	sword := "a black longsword of destruction"
	checks := []struct {
		what string
		char buildChar
		item string
		want []string
	}{
		{"no character", buildChar{}, sword, nil},
		{"class", buildChar{class: "Ranger"}, sword,
			[]string{"A Ranger can't use it (ANTI-RANGER)."}},
		{"hexblades by the rangers' flag", buildChar{class: "Hexblade"},
			sword, []string{"A Hexblade can't use it (ANTI-RANGER)."}},
		{"class it's for", buildChar{class: "Warrior"}, sword, nil},
		{"class group", buildChar{class: "Cleric"}, sword,
			[]string{"A Cleric can't use it (NO-CLERIC)."}},
		{"class group, an", buildChar{class: "Enchanter"}, sword,
			[]string{"An Enchanter can't use it (NO-MAGE)."}},
		{"rogues", buildChar{class: "Rogue"}, "a short sword",
			[]string{"A Rogue can't use it (NO-THIEF)."}},
		{"bards maybe", buildChar{class: "Bard"}, "a short sword",
			[]string{"?A Bard may not be able to use it (NO-THIEF)."}},
		{"usable by its class", buildChar{class: "Wizard"},
			"a phantom blade", nil},
		{"usable by another class", buildChar{class: "Sorcerer"},
			"a phantom blade", []string{"Only for Wizard:Illusionist."}},
		{"an illusionist, a wizard of that subclass",
			buildChar{class: "Illusionist"}, "a phantom blade", nil},
		{"an illusionist, with what's for any wizard",
			buildChar{class: "Illusionist"}, "a wizard's inkwell", nil},
		{"an illusionist, not with another wizard subclass's",
			buildChar{class: "Illusionist"}, "an enchanter's sigil",
			[]string{"Only for Wizard:Enchanter."}},
		{"a wizard, whose subclass the designer doesn't know",
			buildChar{class: "Wizard"}, "an enchanter's sigil", nil},
		{"race and racewar", buildChar{race: "Human"}, "a troll-hide cloak",
			[]string{"A Human can't use it (ANTI-HUMAN ANTI-GOODRACE)."}},
		{"half-orcs by the humans' flag", buildChar{race: "Half-Orc"},
			"a troll-hide cloak",
			[]string{"A Half-Orc can't use it (ANTI-HUMAN)."}},
		{"race on the other side", buildChar{race: "Orc"},
			"a troll-hide cloak", nil},
		{"alignment", buildChar{align: "evil"}, "a suit of lordly plate",
			[]string{"An evil character can't use it (ANTI-EVIL)."}},
		{"sex", buildChar{sex: "male"}, "a pair of goggles",
			[]string{"A male character can't use it (ANTI-MALE)."}},
		{"illithids have no sex", buildChar{race: "Illithid", sex: "male"},
			"a pair of goggles", nil},
		{"all at once", buildChar{class: "Cleric", race: "Human",
			align: "evil", sex: "female"}, "a suit of lordly plate",
			[]string{"An evil character can't use it (ANTI-EVIL)."}},
	}
	for _, c := range checks {
		it, err := itemNamed(db, c.item)
		chkErr(t, err)
		var got []string
		for _, w := range c.char.barred(it) {
			mark := ""
			if w.Maybe {
				mark = "?"
			}
			got = append(got, mark+w.Text)
		}
		chk(t, "barred() "+c.what, c.want, got)
		if maybeOnly := len(c.want) == 0 || c.want[0][0] == '?'; c.char.canUse(it) != maybeOnly {
			t.Errorf("canUse() %s check failed: %v", c.what, !maybeOnly)
		}
	}

	// a character read from a link keeps only what the designer offers
	if c := readChar(url.Values{"class": {"hexblade"}, "race": {"Elf"},
		"align": {"chaotic"}, "sex": {"female"}}); c != (buildChar{
		class: "Hexblade", sex: "female"}) {
		t.Errorf("readChar() check failed: %+v", c)
	}

	// only yuan-ti have tails, and yuan-ti wear nothing in the ears, on the
	// legs or feet, nor a whole-body item
	for race, want := range map[string][]string{
		"": nil,
		"Yuan-Ti": {"ear1: Yuan-ti can't wear anything here.",
			"legs: Yuan-ti can't wear anything here.",
			"feet: Yuan-ti can't wear anything here.",
			"body: Yuan-ti can't wear whole-body items."},
		"Human": {"tail: Only yuan-ti have a tail."},
	} {
		evals := evalSets(t, db, url.Values{"race": {race},
			"a.ear1": {"a tiny mithril stud"}, "a.legs": {"leg irons"},
			"a.feet": {"anklet"}, "a.tail": {"tail ring"},
			"a.arms": {"padded sleeves"}, "b.body": {"quilted robe"}})
		chk(t, "eval() bare slots, "+race, want,
			append(warnings(evals[0]), warnings(evals[1])...))
	}

	// a secondary weapon takes the dual wield skill, which only some classes
	// have, in the game's words; with no class chosen, none is warned of
	for class, want := range map[string][]string{
		"": nil, "Warrior": nil, "Hexblade": nil,
		"Psionicist": {"secondary: A Psionicist lacks the training to use " +
			"two weapons."},
		"Enchanter": {"secondary: An Enchanter lacks the training to use " +
			"two weapons."},
	} {
		evals := evalSets(t, db, url.Values{"class": {class},
			"a.primary": {"a willow wand"}, "a.secondary": {"a willow wand"}})
		chk(t, "eval() a secondary weapon, "+class, want, warnings(evals[0]))
	}
	for _, class := range dualWielders {
		if _, ok := findClass(class); !ok {
			t.Errorf("dualWielders check failed: the designer has no %s", class)
		}
	}
}

func testBuildPaste(t *testing.T, db *sql.DB) {
	list := equipList("You are using:\r\n" +
		"<worn on head>       a horned great helm (magic)\r\n" +
		"<worn in ear>        a tiny mithril stud\r\n" +
		"<worn in ear>        a tiny mithril stud set with a ruby (magic)\r\n" +
		"<worn around wrist>  a bracelet of mages\r\n" +
		"<wielding twohanded> a black longsword of destruction (glowing)\r\n" +
		"<worn on body>       a robe of nothing\r\n" +
		"<worn on tail>       \r\n" +
		"<worn in ear>        a tiny mithril stud\r\n" +
		"\r\n< 751h/751H 991p/991P 140v/140V >\r\n< P: sit > \r\n")
	var got []string
	for _, e := range list {
		got = append(got, e.label+"="+e.name)
	}
	chk(t, "equipList()", []string{"worn on head=a horned great helm",
		"worn in ear=a tiny mithril stud",
		"worn in ear=a tiny mithril stud set with a ruby",
		"worn around wrist=a bracelet of mages",
		"wielding twohanded=a black longsword of destruction",
		"worn on body=a robe of nothing",
		"worn in ear=a tiny mithril stud"}, got)

	// into a set, in place of what it had: a pair's second slot, the first
	// variant, the slot a two-handed weapon goes in, a name not found kept,
	// and a third earring with nowhere to go
	b := readBuild(url.Values{"a.feet": {"irons"}, "a.label": {"mine"},
		"b.head": {"horned"}})
	chkErr(t, b.paste(db, list, "a"))
	got = nil
	for _, s := range buildSlots {
		if name := b.sets[0].names[s.key]; name != "" {
			got = append(got, s.key+"="+name)
		}
	}
	chk(t, "paste()", []string{"head=a horned great helm",
		"ear1=a tiny mithril stud",
		"ear2=a tiny mithril stud set with a ruby", "body=a robe of nothing",
		"wrist1=a bracelet of mages 1",
		"primary=a black longsword of destruction"}, got)
	if b.sets[0].label != "mine" || len(b.sets[1].names) != 1 {
		t.Errorf("paste() check failed: %+v", b.sets)
	}
	// into a new set, the first empty one
	chkErr(t, b.paste(db, list[:1], "new"))
	if len(b.sets) != 3 || b.sets[2].names["head"] != "a horned great helm" {
		t.Errorf("paste() into a new set check failed: %+v", b.sets)
	}

	// two held items, one in each held slot, and an item held in both hands
	// in the first, unless it's a weapon, as a wielded bow shows
	for paste, want := range map[string][]string{
		"<held>               a cursed ember\n" +
			"<held>               a crystal orb (magic)\n": {
			"held=a cursed ember", "held2=a crystal orb"},
		"<held in both hands> a gnarled staff\n": {"held=a gnarled staff"},
		"<held in both hands> a yew longbow (magic)\n": {
			"primary=a yew longbow"},
		"<held in both hands> a bow of nothing\n": {"held=a bow of nothing"},
	} {
		b := readBuild(url.Values{})
		chkErr(t, b.paste(db, equipList(paste), "a"))
		got = nil
		for _, s := range buildSlots {
			if name := b.sets[0].names[s.key]; name != "" {
				got = append(got, s.key+"="+name)
			}
		}
		chk(t, "paste() held items", want, got)
	}
}

// postBuild posts a form to the build designer, returning the status and
// where it redirects.
func postBuild(form url.Values) (int, string) {
	r := httptest.NewRequest("POST", "/build",
		strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	buildHandler(w, r)
	return w.Code, w.Header().Get("Location")
}

func testBuildPost(t *testing.T) {
	stud := "a.ear1=a+tiny+mithril+stud"
	checks := []struct {
		what string
		form url.Values
		want string
	}{
		{"nothing", url.Values{}, "build"},
		{"names found in full", url.Values{"a.head": {"horned"},
			"a.ear1": {"stud"}},
			"build?" + stud + "&a.head=a+horned+great+helm"},
		{"a name not found, as given", url.Values{"a.head": {" a  hat\tof "},
			"a.ear1": {"stud"}}, "build?" + stud + "&a.head=a+hat+of"},
		{"copy into the first empty set", url.Values{"a.ear1": {"stud"},
			"copy": {"a"}}, "build?" + stud + "&b.ear1=a+tiny+mithril+stud"},
		{"copy into a new set", url.Values{"a.ear1": {"stud"},
			"b.ear1": {"ruby"}, "copy": {"b"}}, "build?" + stud +
			"&b.ear1=a+tiny+mithril+stud+set+with+a+ruby" +
			"&c.ear1=a+tiny+mithril+stud+set+with+a+ruby"},
		{"copy with no room", url.Values{"a.ear1": {"stud"},
			"b.ear1": {"stud"}, "c.ear1": {"stud"}, "d.ear1": {"stud"},
			"copy": {"a"}}, "build?" + stud + "&b.ear1=a+tiny+mithril+stud" +
			"&c.ear1=a+tiny+mithril+stud&d.ear1=a+tiny+mithril+stud"},
		{"add a set", url.Values{"a.ear1": {"stud"}, "sets": {"2"},
			"add": {"1"}}, "build?" + stud + "&sets=3"},
		{"no more than four", url.Values{"sets": {"4"}, "add": {"1"}},
			"build?sets=4"},
		{"clear the last set", url.Values{"a.ear1": {"stud"},
			"c.label": {"x"}, "clear": {"c"}}, "build?" + stud},
		{"clear a set, kept", url.Values{"a.ear1": {"stud"},
			"b.ear1": {"stud"}, "c.ear1": {"stud"}, "clear": {"b"}},
			"build?" + stud + "&c.ear1=a+tiny+mithril+stud"},
		{"the character", url.Values{"class": {"ranger"}, "race": {"Elf"},
			"align": {"evil"}, "sex": {"other"}},
			"build?align=evil&class=Ranger"},
		{"a label, cleaned", url.Values{"a.label": {"Tank \x07 gear"}},
			"build?a.label=Tank+gear"},
		{"a pasted list", url.Values{"a.ear1": {"stud"}, "pasteTo": {"b"},
			"paste": {"<worn on head>  a horned great helm\n"}},
			"build?" + stud + "&b.head=a+horned+great+helm"},
		{"a pasted list into a new set", url.Values{"a.ear1": {"stud"},
			"b.ear1": {"stud"}, "pasteTo": {"new"},
			"paste": {"<worn on head>  a horned great helm\n"}},
			"build?" + stud + "&b.ear1=a+tiny+mithril+stud" +
				"&c.head=a+horned+great+helm"},
		{"a long name, cut", url.Values{"a.head": {strings.Repeat("ab ", 50)}},
			"build?a.head=" + strings.TrimSpace(strings.Repeat("ab+", 33)) + "a"},
	}
	for _, c := range checks {
		code, loc := postBuild(c.form)
		if code != http.StatusSeeOther || loc != c.want {
			t.Errorf("buildHandler() POST %s check failed: %d %q, not %q",
				c.what, code, loc, c.want)
		}
	}
}

func testBuildPage(t *testing.T) {
	page := func(query string) (int, string) {
		w := httptest.NewRecorder()
		buildHandler(w, httptest.NewRequest("GET", "/build?"+query, nil))
		return w.Code, w.Body.String()
	}
	code, body := page("")
	if code != http.StatusOK || !strings.Contains(body, "Nothing worn yet") ||
		strings.Contains(body, "Link to this build") ||
		!strings.Contains(body, "Items Updated: 2026-11-01") ||
		!strings.Contains(body,
			`<option value="a" selected="selected">Set A</option>`) {
		t.Errorf("buildHandler() check failed: empty build %d:\n%s", code,
			body)
	}

	code, body = page("class=Cleric&align=evil&a.label=Mine&a.ear1=stud" +
		"&a.primary=stiletto&a.held=ember&a.shield=diamond&b.ear1=ruby")
	if code != http.StatusOK {
		t.Errorf("buildHandler() check failed: GET gave %d", code)
	}
	for _, want := range []string{
		`<option selected="selected">Cleric</option>`,
		`<option value="evil" selected="selected">Evil</option>`,
		`name="a.label" id="a-label" value="Mine"`,
		`name="a.ear1" id="a-ear1" value="a tiny mithril stud"`,
		`name="b.ear1" id="b-ear1" value="a tiny mithril stud set with a ruby"`,
		`data-slot="ear1" aria-label="Set B, worn in ear"`,
		`<div class="stats" title="` + studLong + `">` +
			template.HTMLEscapeString("(EAR) * 0/1 * Wt:0 Val:1p * Skullport (Q)"),
		template.HTMLEscapeString("A Cleric can't use it (NO-CLERIC)."),
		template.HTMLEscapeString("An evil character can't use it (ANTI-EVIL)."),
		"The set needs 3 hands, and a character has two.",
		`<div class="warn">5 problems</div>`,
		`<th scope="colgroup" colspan="3">Totals</th>`,
		`<td>5</td><td>3 <span class="worse">-2</span></td>`,
		`<option value="new" selected="selected">a new set</option>`,
		"Link to this build",
		`a.ear1=a&#43;tiny&#43;mithril&#43;stud&amp;a.held=a&#43;cursed&#43;ember`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("buildHandler() check failed: GET has no %s:\n%s",
				want, body)
		}
	}
	// the page shows a set for each letter up to the last one used
	_, body = page("d.tail=tail+ring")
	if !strings.Contains(body, `class="sets4"`) ||
		strings.Contains(body, "Add a set") {
		t.Errorf("buildHandler() check failed: set D:\n%s", body)
	}
}

func testBuildNames(t *testing.T) {
	names := func(query string) []string {
		w := httptest.NewRecorder()
		namesHandler(w, httptest.NewRequest("GET", "/build/names?"+query,
			nil))
		if w.Code != http.StatusOK || !strings.HasPrefix(
			w.Header().Get("Content-Type"), "application/json") {
			t.Errorf("namesHandler() check failed: %s gave %d %s", query,
				w.Code, w.Header().Get("Content-Type"))
		}
		var found []suggestion
		chkErr(t, json.Unmarshal(w.Body.Bytes(), &found))
		var got []string
		for _, s := range found {
			got = append(got, s.Name+" | "+s.Stats)
		}
		return got
	}
	stud := "a tiny mithril stud | (EAR) * 0/1 * Wt:0 Val:1p * Skullport (Q)"
	short := "a short sword | " +
		"(WIELD) * 1D6 1/1 * !THIEF * Wt:4 Val:1p * Waterdeep (S)"
	sword := "a black longsword of destruction | " +
		strings.TrimPrefix(swordShort, "a black longsword of destruction ")
	checks := []struct {
		query string
		want  []string
	}{
		{"slot=ear1&q=stud", []string{stud, "a tiny mithril stud set with a " +
			"ruby | " + strings.TrimPrefix(rubyShort,
			"a tiny mithril stud set with a ruby ")}},
		{"slot=ear2&q=STUD+ruby", []string{"a tiny mithril stud set with a " +
			"ruby | " + strings.TrimPrefix(rubyShort,
			"a tiny mithril stud set with a ruby ")}},
		{"slot=ear1&q=s", nil},
		{"slot=elbow&q=stud", nil},
		{"q=stud", nil},
		{"slot=neck1&q=stud", nil},
		{"slot=primary&q=sword", []string{short, sword}},
		{"slot=secondary&q=sword", []string{short}},
		{"slot=held&q=staff", []string{"a gnarled staff | " +
			"(HOLD 2H) * Mana:20 * Wt:6 Val:0 * Ashstone"}},
		{"slot=held2&q=staff", nil},
		{"slot=primary&q=sword&class=Ranger", []string{short}},
		{"slot=primary&q=sword&class=Rogue", nil},
		{"slot=primary&q=sword&class=Bard", []string{short, sword}},
		// a secondary weapon only for a class with the dual wield skill
		{"slot=primary&q=sword&class=Cleric", []string{short}},
		{"slot=secondary&q=sword&class=Cleric", nil},
		{"slot=secondary&q=sword&class=Warrior", []string{short}},
		{"slot=finger1&q=evil", []string{"an evil ring | " +
			"#showme pwned; say $cpw @x {y} \\z"}},
		// what a race can wear: a tail only for yuan-ti, which wear nothing
		// in the ears, nor a whole-body item
		{"slot=tail&q=tail+ring&race=Yuan-Ti", []string{"a jeweled tail " +
			"ring | (TAIL) * Hp:10 * Wt:0 Val:0 * Ashstone"}},
		{"slot=tail&q=tail+ring&race=Human", nil},
		{"slot=ear1&q=stud&race=Yuan-Ti", nil},
		{"slot=body&q=quilted", []string{"a quilted robe | (WHOLE-BODY)"}},
		{"slot=body&q=quilted&race=Yuan-Ti", nil},
		{"slot=body&q=jerkin&race=Yuan-Ti", []string{
			"a padded jerkin | (BODY)"}},
	}
	for _, c := range checks {
		chk(t, "namesHandler() "+c.query, c.want, names(c.query))
	}
}

// Test_PageCSP checks that the site's pages hold no inline script or style:
// grokh.com's Content-Security-Policy, default-src 'self', blocks them.
func Test_PageCSP(t *testing.T) {
	pages, err := filepath.Glob(filepath.Join("html", "*.html"))
	chkErr(t, err)
	if len(pages) < 2 {
		t.Errorf("CSP check failed: found %d pages", len(pages))
	}
	inline := regexp.MustCompile(`(?i)<script(\s[^>]*)?>\s*[^<\s]|<style|` +
		`\sstyle\s*=|\son[a-z]+\s*=|javascript:`)
	for _, p := range pages {
		html, err := os.ReadFile(p)
		chkErr(t, err)
		if m := inline.Find(html); m != nil {
			t.Errorf("CSP check failed: %s has inline script or style: %s",
				p, m)
		}
	}
}

// buildDown checks what the designer answers when TorilDB.db can't be read:
// code, both for a build with an item in it and for names to offer, with
// page and names as what they say.
func buildDown(t *testing.T, what string, code int, page, names string) {
	w := httptest.NewRecorder()
	buildHandler(w, httptest.NewRequest("GET", "/build?a.ear1=stud", nil))
	if w.Code != code || strings.TrimSpace(w.Body.String()) != page {
		t.Errorf("buildHandler() check failed: %s gave %d: %q", what,
			w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	namesHandler(w, httptest.NewRequest("GET", "/build/names?slot=ear1&q=stud",
		nil))
	if w.Code != code || strings.TrimSpace(w.Body.String()) != names {
		t.Errorf("namesHandler() check failed: %s gave %d: %q", what,
			w.Code, w.Body.String())
	}
}
