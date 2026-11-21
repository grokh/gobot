package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"maps"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The build designer, /build on the TorilEQ site, dresses a character in
// items from TorilDB.db, one to each slot of the game's equipment list, and
// totals the stats of each set of items side by side. A build is its link:
// the character it's for, and each set's label and the names of its items.
// The page posts each change, and buildHandler answers with a redirect to
// the build's new link, so the address bar always holds the build as it
// stands, and reloading the page can't post a change twice.

// maxSets is the most sets a build compares, A to D. A link to four full
// sets is still well inside the 8 KB nginx allows a request line.
const maxSets = 4

// minSets is how many sets the page shows, filled or not.
const minSets = 2

// maxName and maxLabel are the most bytes kept of a name typed in a slot and
// of a set's label.
const (
	maxName  = 100
	maxLabel = 30
)

// buildSlot is a slot of the designer's character, in the order the game's
// equipment list shows them: key names it in links, label is how the list
// shows it, and slot is the entry in slots whose worn tokens fit it.
type buildSlot struct{ key, label, slot string }

var buildSlots = []buildSlot{
	{"badge", "worn as a badge", "badge"},
	{"head", "worn on head", "head"},
	{"eyes", "worn on eyes", "eyes"},
	{"ear1", "worn in ear", "ear"},
	{"ear2", "worn in ear", "ear"},
	{"face", "worn on face", "face"},
	{"neck1", "worn around neck", "neck"},
	{"neck2", "worn around neck", "neck"},
	{"body", "worn on body", "on_body"},
	{"about", "worn about body", "about"},
	{"quiver", "worn as quiver", "quiver"},
	{"bag", "component bag", "component"},
	{"waist", "worn about waist", "waist"},
	{"arms", "worn on arms", "arms"},
	{"shield", "held as shield", "shield"},
	{"wrist1", "worn around wrist", "wrist"},
	{"wrist2", "worn around wrist", "wrist"},
	{"hands", "worn on hands", "hands"},
	{"finger1", "worn on finger", "finger"},
	{"finger2", "worn on finger", "finger"},
	{"primary", "primary weapon", "wield"},
	{"secondary", "secondary weapon", "wield"},
	{"held", "held", "held"},
	{"legs", "worn on legs", "legs"},
	{"feet", "worn on feet", "feet"},
	{"tail", "worn on tail", "tail"},
}

// equipLabels are the equipment list's names for a weapon or held item that
// takes both hands, with the slot's usual name.
var equipLabels = map[string]string{
	"wielding twohanded": "primary weapon",
	"held in both hands": "held",
}

// handSlots are the slots that take a hand. A character has two, and an item
// that takes both hands counts twice.
var handSlots = []string{"primary", "secondary", "held", "shield"}

// covers are the slots an item covers besides its own: a whole-body item on
// the body covers the arms and legs, and a whole-head item on the head
// covers the eyes and face, as the equipment lists in players' logs show.
// TorilMUD's builder docs say only that they take "all body slots" and "all
// head slots", but players wear earrings with a whole-head helm, and gloves
// and boots with whole-body armor.
var covers = []struct{ slot, flag, what string }{
	{"body", "WHOLE-BODY", "arms"},
	{"body", "WHOLE-BODY", "legs"},
	{"head", "WHOLE-HEAD", "eyes"},
	{"head", "WHOLE-HEAD", "face"},
}

// slotIndex is where the slot with key is in buildSlots, or -1.
func slotIndex(key string) int {
	for i, s := range buildSlots {
		if s.key == key {
			return i
		}
	}
	return -1
}

// worn is the slot's tokens in eq's worn column.
func (s buildSlot) worn() []string {
	sl, _ := findSlot(s.slot)
	return sl.worn
}

// sql is the test for the items that fit the slot, with its arguments: a
// worn token of the slot's, and for the secondary weapon, not two-handed.
func (s buildSlot) sql() (string, []interface{}) {
	test, args := hasToken("worn", s.worn()...)
	if s.key == "secondary" {
		two, targs := twoHandFlag().match()
		test += " AND NOT " + two
		args = append(args, targs...)
	}
	return test, args
}

// fits reports whether the slot can take the item, as sql tests it.
func (s buildSlot) fits(it *buildItem) bool {
	for _, w := range s.worn() {
		if it.wornAs(w) {
			return s.key != "secondary" || !it.twoHanded()
		}
	}
	return false
}

// twoHandFlag is the form's Two Handed flag, whose tokens say an item takes
// both hands.
func twoHandFlag() itemFlag {
	for _, f := range flags {
		if f.abbr == "two_hand" {
			return f
		}
	}
	panic("items.go has no two_hand flag")
}

// buildItem is an item as the designer reads it from eq.
type buildItem struct {
	name  string
	short string // its short stats, less the name they start with
	long  string
	// the tokens of eq's worn, item_flags and affect_flags
	worn, flags, affects []string
	only                 []string // the classes usable_by names, if any
	// its stats by abbr, as stats has them, and resists by "r:" and abbr;
	// those it has none of are left out
	vals   map[string]int
	weight int // what it weighs, eq's wt
}

func hasWord(words []string, w string) bool {
	for _, x := range words {
		if x == w {
			return true
		}
	}
	return false
}

func (it *buildItem) wornAs(token string) bool { return hasWord(it.worn, token) }
func (it *buildItem) flagged(flag string) bool { return hasWord(it.flags, flag) }

// twoHanded reports whether the item takes both hands, as the form's Two
// Handed flag finds it.
func (it *buildItem) twoHanded() bool {
	f := twoHandFlag()
	for _, t := range f.flags {
		if it.flagged(t) {
			return true
		}
	}
	for _, t := range f.worn {
		if it.wornAs(t) {
			return true
		}
	}
	return false
}

// whole reports whether the item has a whole-body or whole-head flag, which
// Kegor writes in worn in place of the slot, and sometimes in item_flags.
func (it *buildItem) whole(flag string) bool {
	return it.wornAs(flag) || it.flagged(flag)
}

// isFor reports whether usable_by names class, as in Hexblade:Hellborne,
// Blackguard, Warrior. What follows a colon narrows it to a specialization,
// which the designer doesn't know, so only the class is checked.
func (it *buildItem) isFor(class string) bool {
	for _, o := range it.only {
		name, _, _ := strings.Cut(o, ":")
		if strings.EqualFold(strings.TrimSpace(name), class) {
			return true
		}
	}
	return false
}

// buildCols is how many columns itemQuery reads before the stats.
const buildCols = 8

// itemQuery is the SELECT for buildItem's columns, as scanItem reads them:
// a column for each of stats, as fstat searches it (acSQL for ac), and then
// one for each of resists.
func itemQuery() string {
	cols := []string{"name", "short_stats", "long_stats", "worn", "item_flags",
		"affect_flags", "usable_by", col("wt")}
	for _, s := range stats {
		cols = append(cols, s.sql())
	}
	for _, r := range resists {
		cols = append(cols, col(r.col))
	}
	return "SELECT " + strings.Join(cols, ", ") + " FROM eq"
}

// number reads a stat. Kegor's columns hold whole numbers or nothing, but a
// few hold an empty string instead of NULL, which counts as none.
func number(s string) int {
	s = strings.TrimSpace(s)
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(math.Round(f))
	}
	return 0
}

// scanItem reads a row of itemQuery.
func scanItem(rows *sql.Rows) (*buildItem, error) {
	vals := make([]sql.NullString, buildCols+len(stats)+len(resists))
	ptrs := make([]interface{}, len(vals))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	it := &buildItem{
		name:    vals[0].String,
		long:    vals[2].String,
		worn:    strings.Fields(vals[3].String),
		flags:   strings.Fields(vals[4].String),
		affects: strings.Fields(vals[5].String),
		weight:  number(vals[7].String),
		vals:    map[string]int{},
	}
	it.short = strings.TrimSpace(strings.TrimPrefix(vals[1].String, it.name))
	for _, o := range strings.Split(vals[6].String, ",") {
		if o = strings.TrimSpace(o); o != "" {
			it.only = append(it.only, o)
		}
	}
	for i, s := range stats {
		if v := number(vals[buildCols+i].String); v != 0 {
			it.vals[s.abbr] = v
		}
	}
	for i, r := range resists {
		if v := number(vals[buildCols+len(stats)+i].String); v != 0 {
			it.vals["r:"+r.abbr] = v
		}
	}
	return it, nil
}

// queryItems reads the items itemQuery finds with the rest of a query.
func queryItems(db *sql.DB, rest string, args ...interface{}) (
	[]*buildItem, error) {
	rows, err := db.Query(itemQuery()+rest, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*buildItem
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// itemNamed reads the item of exactly that name, or nil if there's none.
func itemNamed(db *sql.DB, name string) (*buildItem, error) {
	items, err := queryItems(db, " WHERE name = ?", name)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return items[0], nil
}

// looseItems finds the items for a slot whose names match name loosely, the
// ways stat tries after the exact name: as typed, anywhere in the name, then
// its words in that order with anything between them, then every word in
// any order. Each way's items come before the next way's, closest first,
// which is the shortest name, as stat takes it, and limit caps them all.
// Whatever one way matches, the next matches too, so one read of eq, for the
// last way, finds them all.
func looseItems(db *sql.DB, name string, s buildSlot, limit int) (
	[]*buildItem, error) {
	var ways []nameMatch
	for _, m := range nameMatches(name) {
		if m.op == "LIKE" && len(m.args) > 0 {
			ways = append(ways, m)
		}
	}
	if len(ways) == 0 {
		return nil, nil
	}
	like := func(m nameMatch) (string, []interface{}) {
		var tests []string
		var args []interface{}
		for _, a := range m.args {
			tests = append(tests, "name LIKE ?")
			args = append(args, a)
		}
		return strings.Join(tests, " AND "), args
	}
	where, args := like(ways[len(ways)-1])
	slotTest, slotArgs := s.sql()
	args = append(args, slotArgs...)
	order := "0"
	if len(ways) > 1 {
		order = "CASE"
		for n, m := range ways[:len(ways)-1] {
			test, targs := like(m)
			order += fmt.Sprintf(" WHEN %s THEN %d", test, n)
			args = append(args, targs...)
		}
		order += fmt.Sprintf(" ELSE %d END", len(ways)-1)
	}
	return queryItems(db, " WHERE "+where+" AND "+slotTest+
		" AND short_stats != '' ORDER BY "+order+", length(name), name "+
		"LIMIT ?", append(args, limit)...)
}

// finder finds the items a request's sets name, each name once for each
// kind of slot. It matches at most maxLoose names loosely: a loose match
// reads all of eq, and a link holds up to 104 names, so a link of names that
// find nothing would otherwise cost each request 104 reads of the table.
type finder struct {
	db    *sql.DB
	found map[string]*buildItem
	loose int
}

// maxLoose is the most names a request matches loosely. A post's redirect
// names each item it found in full, so a post from the page has only the
// names typed since the last one, and those that found nothing.
const maxLoose = 30

func newFinder(db *sql.DB) *finder {
	return &finder{db: db, found: map[string]*buildItem{}}
}

// slotItem finds the item a set names for a slot: the item of exactly that
// name, whether it fits the slot or not, or failing that, the closest of
// the items that fit the slot and match the name loosely. It's nil if
// there's none.
func (f *finder) slotItem(name string, s buildSlot) (*buildItem, error) {
	kind := s.slot
	if s.key == "secondary" {
		kind = s.key // no two-handed weapons
	}
	key := kind + "|" + name
	if it, ok := f.found[key]; ok {
		return it, nil
	}
	it, err := itemNamed(f.db, name)
	if err != nil {
		return nil, err
	}
	if it == nil && f.loose < maxLoose {
		f.loose++
		found, err := looseItems(f.db, name, s, 1)
		if err != nil {
			return nil, err
		}
		if len(found) > 0 {
			it = found[0]
		}
	}
	f.found[key] = it
	return it, nil
}

// buildClass is a class the designer checks items against, with the item
// flags that bar it: its group's NO- flag and its own ANTI- flag. They follow
// TorilMUD's builder docs (the Obj-File page of the torilmud/docs wiki), which
// give hexblades the rangers' ANTI- flag and battlechanters the bards'. Kegor
// writes the rogues' as ANTI-THIEF. The docs say NO-THIEF "might" bar bards and
// battlechanters too, so for them it's maybeBar.
type buildClass struct {
	name          string
	bar, maybeBar []string
}

var buildClasses = []buildClass{
	{"Bard", []string{"ANTI-BARD"}, []string{"NO-THIEF"}},
	{"Battlechanter", []string{"ANTI-BARD"}, []string{"NO-THIEF"}},
	{"Blackguard", []string{"NO-WARRIOR", "ANTI-BLACKGUARD"}, nil},
	{"Cleric", []string{"NO-CLERIC", "ANTI-CLERIC"}, nil},
	{"Druid", []string{"NO-CLERIC", "ANTI-DRUID"}, nil},
	{"Elementalist", []string{"NO-MAGE", "ANTI-ELEMENTALIST"}, nil},
	{"Enchanter", []string{"NO-MAGE", "ANTI-ENCHANTER"}, nil},
	{"Hexblade", []string{"NO-WARRIOR", "ANTI-RANGER"}, nil},
	{"Illusionist", []string{"NO-MAGE", "ANTI-ILLUSIONIST"}, nil},
	{"Invoker", []string{"NO-MAGE", "ANTI-INVOKER"}, nil},
	{"Lich", []string{"NO-MAGE", "ANTI-LICH"}, nil},
	{"Necromancer", []string{"NO-MAGE", "ANTI-NECROMANCER"}, nil},
	{"Paladin", []string{"NO-WARRIOR", "ANTI-PALADIN"}, nil},
	{"Psionicist", []string{"NO-MAGE", "ANTI-PSIONICIST"}, nil},
	{"Ranger", []string{"NO-WARRIOR", "ANTI-RANGER"}, nil},
	{"Rogue", []string{"NO-THIEF", "ANTI-THIEF"}, nil},
	{"Shaman", []string{"NO-CLERIC", "ANTI-SHAMAN"}, nil},
	{"Sorcerer", []string{"NO-MAGE"}, nil},
	{"Warrior", []string{"NO-WARRIOR", "ANTI-WARRIOR"}, nil},
	{"Wizard", []string{"NO-MAGE"}, nil},
}

// buildRace is a race the designer checks items against: its side of the
// racewar, and the item flags that bar it. Half-orcs are on neither side,
// and the builder docs say they can use either side's items, but not an
// item barred to humans or to orcs.
type buildRace struct {
	name, side string
	bar        []string
}

var buildRaces = []buildRace{
	{"Barbarian", "good", []string{"ANTI-BARBARIAN"}},
	{"Drow Elf", "evil", []string{"ANTI-DROWELF"}},
	{"Duergar", "evil", []string{"ANTI-DUERGAR"}},
	{"Gnome", "good", []string{"ANTI-GNOME"}},
	{"Half-Elf", "good", []string{"ANTI-HALFELF"}},
	{"Half-Orc", "", []string{"ANTI-HALFORC", "ANTI-HUMAN", "ANTI-ORC"}},
	{"Halfling", "good", []string{"ANTI-HALFLING"}},
	{"Human", "good", []string{"ANTI-HUMAN"}},
	{"Illithid", "evil", []string{"ANTI-ILLITHID"}},
	{"Moon Elf", "good", []string{"ANTI-GREYELF"}},
	{"Ogre", "evil", []string{"ANTI-OGRE"}},
	{"Orc", "evil", []string{"ANTI-ORC"}},
	{"Shield Dwarf", "good", []string{"ANTI-DWARF"}},
	{"Troll", "evil", []string{"ANTI-TROLL"}},
	{"Yuan-Ti", "evil", []string{"ANTI-YUANTI"}},
}

// sideFlags bar a side of the racewar.
var sideFlags = map[string]string{
	"good": "ANTI-GOODRACE",
	"evil": "ANTI-EVILRACE",
}

// aligns and sexes are the alignments and sexes the designer offers, with
// the flags that bar them. Illithids have no sex, so neither flag bars them:
// Kegor's short stats call an item with both ILLITHID-ONLY.
var (
	aligns = []Option{{"good", "Good"}, {"neutral", "Neutral"},
		{"evil", "Evil"}}
	alignFlags = map[string]string{"good": "ANTI-GOOD",
		"neutral": "ANTI-NEUTRAL", "evil": "ANTI-EVIL"}
	sexes     = []Option{{"male", "Male"}, {"female", "Female"}}
	sexFlags  = map[string]string{"male": "ANTI-MALE", "female": "ANTI-FEMALE"}
	noSexRace = "Illithid"
)

// tailRace is the one race with a tail to wear an item on. It has a tail in
// place of legs and feet, so it wears nothing on them, nor a whole-body item,
// which covers the legs, and it wears no earrings: the yuan-ti whose
// equipment players' logs list wear none of them.
const tailRace = "Yuan-Ti"

// tailRaceBare are the slots tailRace has nothing to wear an item on.
var tailRaceBare = []string{"ear1", "ear2", "legs", "feet"}

// bareSlots are the slots a character of race has nothing to wear an item
// on, by key: the tail, unless it's a yuan-ti, and a yuan-ti's tailRaceBare.
// For a race left blank, they're every race's, so that what a fill puts in a
// set suits every race. The designer warns of neither (see wears).
func bareSlots(race string) []string {
	switch race {
	case tailRace:
		return tailRaceBare
	case "":
		return append([]string{"tail"}, tailRaceBare...)
	}
	return []string{"tail"}
}

// coversBare reports whether a whole item worn in the slot with key would
// cover one of the bare slots, so that it can't be worn.
func coversBare(bare []string, key string) bool {
	for _, c := range covers {
		if c.slot == key && hasWord(bare, c.what) {
			return true
		}
	}
	return false
}

// wears reports whether the character has what it takes to wear the item in
// slot s, as far as its race is chosen: the slot, and any slot the item
// would cover.
func (c buildChar) wears(s buildSlot, it *buildItem) bool {
	if c.race == "" {
		return true
	}
	bare := bareSlots(c.race)
	return !hasWord(bare, s.key) &&
		!(it.whole(wholeFlag(s.key)) && coversBare(bare, s.key))
}

// noPlayer are the flags Kegor gives items no player can use.
var noPlayer = []string{"ANTI-PLAYER", "NO-PLAYER", "NO-PC"}

func findClass(name string) (buildClass, bool) {
	for _, c := range buildClasses {
		if strings.EqualFold(c.name, name) {
			return c, true
		}
	}
	return buildClass{}, false
}

func findRace(name string) (buildRace, bool) {
	for _, r := range buildRaces {
		if strings.EqualFold(r.name, name) {
			return r, true
		}
	}
	return buildRace{}, false
}

// buildChar is who a build is for, as far as items care: each part is ""
// when it isn't given, and then nothing about it bars an item.
type buildChar struct{ class, race, align, sex string }

// readChar reads the character from a form or link, leaving out any part
// that isn't one the designer offers.
func readChar(v url.Values) buildChar {
	var c buildChar
	if cl, ok := findClass(v.Get("class")); ok {
		c.class = cl.name
	}
	if r, ok := findRace(v.Get("race")); ok {
		c.race = r.name
	}
	if _, ok := alignFlags[v.Get("align")]; ok {
		c.align = v.Get("align")
	}
	if _, ok := sexFlags[v.Get("sex")]; ok {
		c.sex = v.Get("sex")
	}
	return c
}

// warning is something wrong with an item where a set has it. A maybe is one
// the builder docs aren't sure of.
type warning struct {
	Text  string
	Maybe bool
}

// article is "a" or "an" for a word, by its first letter.
func article(word string) string {
	if word != "" && strings.ContainsRune("AEIOUaeiou", rune(word[0])) {
		return "an"
	}
	return "a"
}

// barred is why the character can't use the item, or may not: a warning for
// each part of the character that one of its flags bars.
func (c buildChar) barred(it *buildItem) []warning {
	var ws []warning
	bar := func(who string, maybe bool, flags ...string) {
		var hit []string
		for _, f := range flags {
			if f != "" && it.flagged(f) {
				hit = append(hit, f)
			}
		}
		if len(hit) == 0 {
			return
		}
		can := " can't use it ("
		if maybe {
			can = " may not be able to use it ("
		}
		who = strings.ToUpper(who[:1]) + who[1:]
		ws = append(ws, warning{who + can + strings.Join(hit, " ") + ").", maybe})
	}
	for _, f := range noPlayer {
		if it.flagged(f) {
			ws = append(ws, warning{Text: "No player can use it (" + f + ")."})
		}
	}
	if cl, ok := findClass(c.class); ok {
		who := article(cl.name) + " " + cl.name
		bar(who, false, cl.bar...)
		bar(who, true, cl.maybeBar...)
		if len(it.only) > 0 && !it.isFor(cl.name) {
			ws = append(ws, warning{Text: "Only for " +
				strings.Join(it.only, ", ") + "."})
		}
	}
	if r, ok := findRace(c.race); ok {
		bar(article(r.name)+" "+r.name, false,
			append(r.bar, sideFlags[r.side])...)
	}
	if f, ok := alignFlags[c.align]; ok {
		bar(article(c.align)+" "+c.align+" character", false, f)
	}
	if f, ok := sexFlags[c.sex]; ok && c.race != noSexRace {
		bar("a "+c.sex+" character", false, f)
	}
	return ws
}

// canUse reports whether nothing certain bars the character from the item.
func (c buildChar) canUse(it *buildItem) bool {
	for _, w := range c.barred(it) {
		if !w.Maybe {
			return false
		}
	}
	return true
}

// buildSet is one set of a build: its label, and the name given for each of
// its slots, by key.
type buildSet struct {
	label string
	names map[string]string
}

func (s buildSet) empty() bool { return s.label == "" && len(s.names) == 0 }

// build is a whole build: the character, the sets the page shows, what the
// page's Fill (best.go) is asked for, and the essentials the set it last
// filled lacks, if it needs them, which only the link from that fill tells.
type build struct {
	char  buildChar
	sets  []buildSet
	best  bestOpts
	lacks []string
}

// setKey is the letter that names the nth set in links, a to d.
func setKey(n int) string { return string(rune('a' + n)) }

// setIndex is which set a letter names, or -1.
func setIndex(key string) int {
	for n := 0; n < maxSets; n++ {
		if key == setKey(n) {
			return n
		}
	}
	return -1
}

// field cleans a value typed on the page: control characters and runs of
// spaces become one space, and it's cut to max bytes.
func field(s string, max int) string {
	s = strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)), " ")
	if len(s) > max {
		cut := max
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = strings.TrimSpace(s[:cut])
	}
	return s
}

// readBuild reads a build from a form or a link: the character, what a fill
// is asked for and the essentials it lacked, then each set's label and the
// name in each of its slots, from a.label, a.head, b.head and so on. The
// page shows as many sets as sets asks, at least minSets, and at least
// enough for the last set with anything in it.
func readBuild(v url.Values) build {
	b := build{char: readChar(v), best: readBest(v)}
	for _, e := range essentials {
		if b.best.resists && hasWord(v["lacks"], e) {
			b.lacks = append(b.lacks, e)
		}
	}
	all := make([]buildSet, maxSets)
	last := -1
	for n := range all {
		k := setKey(n)
		all[n].label = field(v.Get(k+".label"), maxLabel)
		all[n].names = map[string]string{}
		for _, s := range buildSlots {
			if name := field(v.Get(k+"."+s.key), maxName); name != "" {
				all[n].names[s.key] = name
			}
		}
		if !all[n].empty() {
			last = n
		}
	}
	shown, _ := strconv.Atoi(v.Get("sets"))
	b.sets = all[:min(max(shown, last+1, minSets), maxSets)]
	return b
}

// values is the build as link values, which readBuild reads back: only
// what's filled in, what a fill is asked for unless it's the default, the
// essentials the set it filled lacks, and sets only when it shows more sets
// than it would without it.
func (b build) values() url.Values {
	v := url.Values{}
	for k, val := range map[string]string{"class": b.char.class,
		"race": b.char.race, "align": b.char.align, "sex": b.char.sex} {
		if val != "" {
			v.Set(k, val)
		}
	}
	b.best.addValues(v)
	for _, e := range b.lacks {
		v.Add("lacks", e)
	}
	last := -1
	for n, s := range b.sets {
		k := setKey(n)
		if s.label != "" {
			v.Set(k+".label", s.label)
		}
		for key, name := range s.names {
			v.Set(k+"."+key, name)
		}
		if !s.empty() {
			last = n
		}
	}
	if len(b.sets) > max(last+1, minSets) {
		v.Set("sets", strconv.Itoa(len(b.sets)))
	}
	return v
}

// link is the build's address, relative to the page's own, which works
// behind nginx's /torileq/ as well as without it.
func (b build) link() string {
	if q := b.values().Encode(); q != "" {
		return "build?" + q
	}
	return "build"
}

// emptySet is the first set other than not with nothing in it, or a new one
// if every set has something and there's room for another. It's -1 if
// there's no room.
func (b *build) emptySet(not int) int {
	for n, s := range b.sets {
		if n != not && s.empty() {
			return n
		}
	}
	if len(b.sets) < maxSets {
		b.sets = append(b.sets, buildSet{names: map[string]string{}})
		return len(b.sets) - 1
	}
	return -1
}

// target is the set a pasted list or a fill goes in: the set to names, or
// else the first empty one, which is a new one if there's room for it. It's
// -1 if there's none.
func (b *build) target(to string) int {
	if n := setIndex(to); n >= 0 && n < len(b.sets) {
		return n
	}
	return b.emptySet(-1)
}

// act does what the button a post was sent by asks for: add a set, copy one
// into the first empty set, clear one, which takes the last set away if the
// page shows more than minSets, or fill one with the best items for a stat.
// Any equipment list pasted in goes into the set pasteTo names, or the first
// empty one. Only a fill's own link notes the essentials its set lacks.
func (b *build) act(db *sql.DB, v url.Values) error {
	b.lacks = nil
	switch {
	case v.Get("fill") != "":
		if err := b.fill(db, v); err != nil {
			return err
		}
	case v.Get("add") != "":
		if len(b.sets) < maxSets {
			b.sets = append(b.sets, buildSet{names: map[string]string{}})
		}
	case v.Get("copy") != "":
		from := setIndex(v.Get("copy"))
		if from < 0 || from >= len(b.sets) {
			break
		}
		if to := b.emptySet(from); to >= 0 {
			b.sets[to].names = maps.Clone(b.sets[from].names)
		}
	case v.Get("clear") != "":
		n := setIndex(v.Get("clear"))
		if n < 0 || n >= len(b.sets) {
			break
		}
		b.sets[n] = buildSet{names: map[string]string{}}
		if n == len(b.sets)-1 && n >= minSets {
			b.sets = b.sets[:n]
		}
	}
	if list := equipList(v.Get("paste")); len(list) > 0 {
		return b.paste(db, list, v.Get("pasteTo"))
	}
	return nil
}

// equipLine is a line of the game's equipment list, or of what look shows a
// character wearing: the slot, then the item, as in "<worn on head>
// a star-metal chainmail hood (magic)".
var equipLine = regexp.MustCompile(`^<([a-z][a-z ]*)>\s*\S`)

// equipped is an item a pasted equipment list shows, and the slot it's in.
type equipped struct{ label, name string }

// equipList reads the items out of a pasted equipment list, by the slot
// each is in, leaving out any line that isn't one of the list's.
func equipList(txt string) []equipped {
	var list []equipped
	for _, line := range strings.Split(txt, "\n") {
		line = strings.TrimSpace(line)
		m := equipLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if name := lineName(line); name != "" {
			list = append(list, equipped{strings.TrimSpace(m[1]), name})
		}
	}
	return list
}

// paste puts the items of a pasted equipment list in a set, in place of what
// it had: in the set to names, or else the first empty one. Each goes in its
// slot, or the second of a pair once the first is taken, by its name in eq if
// pastedItem finds it there. A set with no room is left alone.
func (b *build) paste(db *sql.DB, list []equipped, to string) error {
	n := b.target(to)
	if n < 0 {
		return nil
	}
	names := map[string]string{}
	for _, e := range list {
		label := e.label
		if l, ok := equipLabels[label]; ok {
			label = l
		}
		for _, s := range buildSlots {
			if s.label != label || names[s.key] != "" {
				continue
			}
			name, err := pastedItem(db, e.name)
			if err != nil {
				return err
			}
			if name == "" {
				name = e.name
			}
			names[s.key] = field(name, maxName)
			break
		}
	}
	b.sets[n].names = names
	return nil
}

// settle puts each item's name in eq in place of the name given for it, so
// the link a post redirects to holds each item by its full name. A name that
// finds nothing is kept as it was given.
func (b *build) settle(f *finder) error {
	for _, set := range b.sets {
		for _, s := range buildSlots {
			name := set.names[s.key]
			if name == "" {
				continue
			}
			it, err := f.slotItem(name, s)
			if err != nil {
				return err
			}
			if it != nil {
				set.names[s.key] = it.name
			}
		}
	}
	return nil
}

// setEval is a set's items, a slot at a time as buildSlots has them, each
// slot's warnings, and the set's totals.
type setEval struct {
	items []*buildItem
	warns [][]warning
	// totals: by stat abbr and resist, as buildItem's vals; the effects the
	// items give, by affect_flags token; how many items; what they weigh
	vals    map[string]int
	effects map[string]bool
	count   int
	weight  int
}

// problems is how many of the set's warnings aren't maybes.
func (e setEval) problems() int {
	n := 0
	for _, ws := range e.warns {
		for _, w := range ws {
			if !w.Maybe {
				n++
			}
		}
	}
	return n
}

// eval finds a set's items and what's wrong with them, and totals them: an
// item that doesn't fit its slot or that the character can't use still
// counts, but is warned of.
func (b build) eval(f *finder, set buildSet) (setEval, error) {
	e := setEval{
		items:   make([]*buildItem, len(buildSlots)),
		warns:   make([][]warning, len(buildSlots)),
		vals:    map[string]int{},
		effects: map[string]bool{},
	}
	warn := func(n int, text string) {
		e.warns[n] = append(e.warns[n], warning{Text: text})
	}
	for n, s := range buildSlots {
		name := set.names[s.key]
		if name == "" {
			continue
		}
		it, err := f.slotItem(name, s)
		if err != nil {
			return e, err
		}
		if it == nil {
			warn(n, "No item by that name fits this slot.")
			continue
		}
		e.items[n] = it
		switch {
		case s.fits(it):
		case s.key == "secondary" && it.wornAs("WIELD"):
			warn(n, "It takes both hands, so it can only be the primary weapon.")
		case len(it.worn) == 0 || it.wornAs("NOBITS"):
			warn(n, "TorilEQ lists it as not worn at all.")
		default:
			warn(n, "TorilEQ lists it as worn "+strings.Join(it.worn, " ")+
				", not here.")
		}
		e.warns[n] = append(e.warns[n], b.char.barred(it)...)
		switch bare := bareSlots(b.char.race); {
		case b.char.wears(s, it):
		case s.key == "tail":
			warn(n, "Only yuan-ti have a tail.")
		case hasWord(bare, s.key):
			warn(n, "Yuan-ti can't wear anything here.")
		default:
			warn(n, "Yuan-ti can't wear "+
				strings.ToLower(wholeFlag(s.key))+" items.")
		}
	}

	hands := 0
	for _, k := range handSlots {
		if it := e.items[slotIndex(k)]; it != nil {
			hands++
			if it.twoHanded() {
				hands++
			}
		}
	}
	if hands > 2 {
		for _, k := range handSlots {
			if n := slotIndex(k); e.items[n] != nil {
				warn(n, fmt.Sprintf("The set needs %d hands, and a character "+
					"has two.", hands))
			}
		}
	}
	for _, c := range covers {
		it, n := e.items[slotIndex(c.slot)], slotIndex(c.what)
		if it != nil && it.whole(c.flag) && e.items[n] != nil {
			warn(n, "The "+strings.ToLower(c.flag)+" item "+
				buildSlots[slotIndex(c.slot)].label+" covers this slot.")
		}
	}

	for _, it := range e.items {
		if it == nil {
			continue
		}
		e.count++
		e.weight += it.weight
		for k, v := range it.vals {
			e.vals[k] += v
		}
		for _, a := range it.affects {
			if a != "NOBITS" {
				e.effects[a] = true
			}
		}
	}
	return e, nil
}

// statDirs are the stats whose totals aren't better for being more: the
// saving throws are better less, and age, height, weight and karma are
// neither better nor worse.
var statDirs = map[string]int{"svsp": -1, "svbr": -1, "svpar": -1,
	"svpet": -1, "svrod": -1, "age": 0, "wt": 0, "ht": 0, "karma": 0}

// badEffects are effects an item is worse for giving.
var badEffects = map[string]bool{"BLINDNESS": true, "SLOW": true}

// Diff is how a set's total differs from the first set's, and whether that's
// better or worse, as a class for style.css.
type Diff struct{ Text, Class string }

// TotalCell is a set's total for one row of the totals.
type TotalCell struct {
	Value string
	Diffs []Diff
}

// TotalRow is a row of the totals: what it totals, and each set's total.
type TotalRow struct {
	Label string
	Cells []TotalCell
}

// effectName is the name the search form gives an affect_flags token, or the
// token itself for one it doesn't list.
func effectName(token string) string {
	for _, e := range effects {
		if e.token == token {
			return e.display
		}
	}
	return token
}

// totalRows are the rows of the totals: how many items each set has, then
// each stat and resist any set has, in the order the search form lists them,
// the effects, and what the items weigh. A set after the first also shows
// how it differs from the first, unless one of them is empty. An empty set
// shows nothing.
func totalRows(evals []setEval) []TotalRow {
	var rows []TotalRow
	first := evals[0]
	differs := func(n int, e setEval) bool {
		return n > 0 && first.count > 0 && e.count > 0
	}
	num := func(label, unit string, dir int, get func(setEval) int) {
		row := TotalRow{Label: label}
		some := false
		for n, e := range evals {
			var cell TotalCell
			if e.count > 0 {
				v := get(e)
				some = some || v != 0
				cell.Value = strconv.Itoa(v) + unit
				if d := v - get(first); differs(n, e) && d != 0 {
					class := ""
					switch {
					case d*dir > 0:
						class = "better"
					case d*dir < 0:
						class = "worse"
					}
					cell.Diffs = []Diff{{fmt.Sprintf("%+d", d) + unit, class}}
				}
			}
			row.Cells = append(row.Cells, cell)
		}
		if some {
			rows = append(rows, row)
		}
	}

	num("Items", "", 0, func(e setEval) int { return e.count })
	for _, s := range stats {
		dir, ok := statDirs[s.abbr]
		if !ok {
			dir = 1
		}
		abbr := s.abbr
		num(s.display, "", dir, func(e setEval) int { return e.vals[abbr] })
	}
	for _, r := range resists {
		key := "r:" + r.abbr
		num(r.display+" resist", "%", 1,
			func(e setEval) int { return e.vals[key] })
	}

	// effects, in the order the search form lists them, then any it doesn't
	var tokens, others []string
	for _, ef := range effects {
		tokens = append(tokens, ef.token)
	}
	for _, e := range evals {
		for t := range e.effects {
			if !hasWord(tokens, t) && !hasWord(others, t) {
				others = append(others, t)
			}
		}
	}
	sort.Strings(others)
	tokens = append(tokens, others...)
	row := TotalRow{Label: "Effects"}
	some := false
	for n, e := range evals {
		var cell TotalCell
		var names []string
		for _, t := range tokens {
			has := e.effects[t]
			if has {
				names = append(names, effectName(t))
			}
			if !differs(n, e) || has == first.effects[t] {
				continue
			}
			sign, better := "+", has != badEffects[t]
			if !has {
				sign = "-"
			}
			class := "worse"
			if better {
				class = "better"
			}
			cell.Diffs = append(cell.Diffs, Diff{sign + effectName(t), class})
		}
		some = some || len(names) > 0
		cell.Value = strings.Join(names, ", ")
		row.Cells = append(row.Cells, cell)
	}
	if some {
		rows = append(rows, row)
	}

	num("Equipment weight", "", -1, func(e setEval) int { return e.weight })
	return rows
}

// SetHead is the heading of a set's column.
type SetHead struct {
	Key, Name, Label string // a, Set A, and the label it's been given
	Problems         int
}

// SlotCell is a set's slot: its box, holding the name of the item found or
// the name given, and the item's stats and warnings.
type SlotCell struct {
	Field, ID, Slot, Aria, Value string
	Stats, Long                  string
	Warnings                     []warning
}

// SlotRow is a slot, as the equipment list labels it, and each set's cell.
type SlotRow struct {
	Label string
	Cells []SlotCell
}

// Check is a checkbox the page shows, and whether it's checked.
type Check struct {
	Abbr, Disp string
	Checked    bool
}

// BuildPage is what html/build.html shows.
type BuildPage struct {
	Date, Link              string
	Class, Race, Align, Sex string
	Classes, Races          []string
	Aligns, Sexes           []Option
	Sets                    []SetHead
	Slots                   []SlotRow
	Totals                  []TotalRow
	Cols                    int // the table's columns, the slots' and the sets'
	CanAdd                  bool
	PasteTo                 []Option // the sets a pasted list or a fill can go in
	PasteDefault            string   // the first empty one, or new
	// what a fill is asked for, and what the page offers
	Best, Then, Hands   string
	BestStats, HandWays []Option
	LeaveOuts           []Check
	Resists             bool
	Lacks               string // the essentials the set it filled lacks
}

// itemsUpdated is the date of the newest item in TorilDB.db.
func itemsUpdated(db *sql.DB) string {
	var date sql.NullString
	ChkErr(db.QueryRow("SELECT MAX(date) FROM eq").Scan(&date))
	return date.String
}

// page is the build designer's page for the build.
func (b build) page(db *sql.DB) (BuildPage, error) {
	p := BuildPage{
		Date: itemsUpdated(db), Class: b.char.class, Race: b.char.race,
		Align: b.char.align, Sex: b.char.sex, Aligns: aligns, Sexes: sexes,
		Cols: len(b.sets) + 1, CanAdd: len(b.sets) < maxSets,
		Best: b.best.stat, Then: b.best.then, Hands: b.best.hands,
		Resists: b.best.resists, Lacks: andList(b.lacks),
	}
	for _, c := range buildClasses {
		p.Classes = append(p.Classes, c.name)
	}
	for _, r := range buildRaces {
		p.Races = append(p.Races, r.name)
	}
	for _, s := range bestStats() {
		p.BestStats = append(p.BestStats, Option{s.abbr, s.display})
	}
	for _, h := range handChoices {
		p.HandWays = append(p.HandWays, Option{h.abbr, h.display})
	}
	for _, l := range leaveOuts {
		p.LeaveOuts = append(p.LeaveOuts,
			Check{l.abbr, l.display, hasWord(b.best.leave, l.abbr)})
	}

	// the link holds each item by its name in eq, as a post's redirect does
	settled := build{char: b.char, best: b.best}
	var evals []setEval
	f := newFinder(db)
	for n, set := range b.sets {
		e, err := b.eval(f, set)
		if err != nil {
			return p, err
		}
		evals = append(evals, e)
		names := map[string]string{}
		maps.Copy(names, set.names)
		for i, it := range e.items {
			if it != nil {
				names[buildSlots[i].key] = it.name
			}
		}
		settled.sets = append(settled.sets, buildSet{set.label, names})

		name := "Set " + strings.ToUpper(setKey(n))
		p.Sets = append(p.Sets, SetHead{setKey(n), name, set.label,
			e.problems()})
		p.PasteTo = append(p.PasteTo, Option{setKey(n), name})
		if p.PasteDefault == "" && set.empty() {
			p.PasteDefault = setKey(n)
		}
	}
	if p.CanAdd {
		p.PasteTo = append(p.PasteTo, Option{"new", "a new set"})
		if p.PasteDefault == "" {
			p.PasteDefault = "new"
		}
	}
	p.Link = settled.link()

	for i, s := range buildSlots {
		row := SlotRow{Label: s.label}
		for n, set := range b.sets {
			k := setKey(n)
			c := SlotCell{Field: k + "." + s.key, ID: k + "-" + s.key,
				Slot: s.key, Aria: p.Sets[n].Name + ", " + s.label,
				Value: set.names[s.key], Warnings: evals[n].warns[i]}
			if it := evals[n].items[i]; it != nil {
				c.Value, c.Stats, c.Long = it.name, it.short, it.long
			}
			row.Cells = append(row.Cells, c)
		}
		p.Slots = append(p.Slots, row)
	}
	p.Totals = totalRows(evals)
	return p, nil
}

// buildTmpl is the build designer's page, which loadPage loads.
var buildTmpl *template.Template

// webRecover answers a request whose handler panicked, as fatal does, with a
// 500 that says sorry, rather than let one bad request take the whole site
// down. Defer it.
func webRecover(w http.ResponseWriter, r *http.Request, sorry string) {
	if err := recover(); err != nil {
		log.Printf("web: %s %s failed: %v", r.Method, r.URL.Path, err)
		http.Error(w, sorry, http.StatusInternalServerError)
	}
}

// buildSorry is the designer's answer to a request that failed.
const buildSorry = "Sorry, that build failed. Please try again."

// maxPost is the most a post to the designer may send: a pasted equipment
// list is a few KB at most. nginx allows 50 KB.
const maxPost = 64 << 10

// buildHandler serves the build designer. A GET shows the build its link
// holds. A post changes the build as the page's buttons ask and redirects
// to its new link, with each item named in full.
func buildHandler(w http.ResponseWriter, r *http.Request) {
	defer webRecover(w, r, buildSorry)
	if _, err := os.Stat(itemDBFile); err != nil {
		log.Printf("web: %s: %v", itemDBFile, err)
		http.Error(w, itemsDown, http.StatusServiceUnavailable)
		return
	}
	db, err := OpenItemDB()
	ChkErr(err)
	defer db.Close()

	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, maxPost)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "That build couldn't be read.",
				http.StatusBadRequest)
			return
		}
		b := readBuild(r.PostForm)
		ChkErr(b.act(db, r.PostForm))
		ChkErr(b.settle(newFinder(db)))
		// set by hand: http.Redirect would make it absolute, as /build,
		// which behind nginx's /torileq/ is another site's page
		w.Header().Set("Location", b.link())
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	p, err := readBuild(r.URL.Query()).page(db)
	ChkErr(err)
	var page bytes.Buffer
	ChkErr(buildTmpl.Execute(&page, p))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page.WriteTo(w)
}

// suggestLimit is the most names the designer offers for a slot at once.
const suggestLimit = 15

// suggestion is a name the designer offers as a slot's box is typed in, with
// the item's short stats.
type suggestion struct {
	Name  string `json:"name"`
	Stats string `json:"stats"`
}

// suggest finds the items to offer for a slot as q is typed in it, matched
// loosely, as a typed name is found, and closest first: those that fit the
// slot, that the character can wear there, and that nothing certain bars the
// character from.
func suggest(db *sql.DB, s buildSlot, q string, c buildChar) (
	[]suggestion, error) {
	found, err := looseItems(db, q, s, 100)
	if err != nil {
		return nil, err
	}
	names := []suggestion{}
	for _, it := range found {
		if len(names) < suggestLimit && c.wears(s, it) && c.canUse(it) {
			names = append(names, suggestion{it.name, it.short})
		}
	}
	return names, nil
}

// namesHandler answers build.js's ask for the names to offer for a slot
// (?slot=head&q=circlet, with the character's class, race, align and sex
// if they're set) as JSON: a list of {name, stats}, empty for a slot it
// doesn't know or fewer than 2 letters.
func namesHandler(w http.ResponseWriter, r *http.Request) {
	defer webRecover(w, r, buildSorry)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := field(r.FormValue("q"), maxName)
	n := slotIndex(r.FormValue("slot"))
	if n < 0 || utf8.RuneCountInString(q) < 2 {
		w.Write([]byte("[]\n"))
		return
	}
	db, err := OpenItemDB()
	if err != nil {
		log.Printf("web: %s: %v", itemDBFile, err)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("[]\n"))
		return
	}
	defer db.Close()
	names, err := suggest(db, buildSlots[n], q, readChar(r.Form))
	ChkErr(err)
	ChkErr(json.NewEncoder(w).Encode(names))
}
