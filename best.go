package main

import (
	"database/sql"
	"maps"
	"net/url"
	"slices"
	"strings"
)

// Fill, on the build designer's page, fills a set with the items that give
// the build's character the most of a stat or resist, or the least of a
// saving throw: the most any set of items can give, not the best item for
// each slot taken one at a time. A whole-body suit is weighed against a body
// item with arms and legs, a whole-head helm against a head item with eyes
// and face, and each way to fill two hands against the others. A second stat
// breaks ties and fills the slots the first leaves empty, and then the
// lighter items win. A slot that would add nothing is left empty.
//
// A fill uses only items the character can use with no warning at all, not
// even a maybe, and none a player can't have: those Kegor marks out of the
// game (load code !) or no take (load code X, or NOTAKE). A part of the
// character the page leaves blank stands for every class, race, alignment or
// sex the page offers, so a set suits them all: the designer can't tell which
// of them the game lets go together, so it never picks one. Players wear two
// of the same ring, earring or bracer, as their logs show, so a pair's two
// slots can hold the same item.

// bestStat is a stat or resist a set can be filled for: the name the page
// posts, its key in buildItem's vals, its name on the page, its SQL, and
// which way is better, 1 more or -1 less.
type bestStat struct {
	abbr, key, display, sql string
	dir                     int
}

// bestStats are what the page offers to fill a set for, in the search form's
// order: each stat that's better more or better less, then the resists.
func bestStats() []bestStat {
	var list []bestStat
	for _, s := range stats {
		dir, ok := statDirs[s.abbr]
		if !ok {
			dir = 1
		}
		if dir != 0 {
			list = append(list, bestStat{s.abbr, s.abbr, s.display, s.sql(),
				dir})
		}
	}
	for _, r := range resists {
		list = append(list, bestStat{r.abbr, "r:" + r.abbr,
			r.display + " resist", col(r.col), 1})
	}
	return list
}

func findBestStat(abbr string) (bestStat, bool) {
	for _, s := range bestStats() {
		if strings.EqualFold(s.abbr, abbr) {
			return s, true
		}
	}
	return bestStat{}, false
}

// handUse is what a fill may put in a hand slot: nothing, an item that takes
// one hand, one that takes one hand or both, or only one that takes both.
type handUse int

const (
	noHand handUse = iota
	oneHand
	anyHands
	bothHands
)

// handChoice is a way the page offers to fill the hands, by what each hand
// slot may hold. A slot it doesn't name stays empty.
type handChoice struct {
	abbr, display string
	use           map[string]handUse
}

var handChoices = []handChoice{
	{"any", "any way", map[string]handUse{"primary": anyHands,
		"secondary": oneHand, "held": anyHands, "shield": anyHands}},
	{"shield", "a weapon and a shield",
		map[string]handUse{"primary": oneHand, "shield": oneHand}},
	{"two", "two weapons",
		map[string]handUse{"primary": oneHand, "secondary": oneHand}},
	{"twohanded", "a two-handed weapon",
		map[string]handUse{"primary": bothHands}},
	{"held", "a weapon and a held item",
		map[string]handUse{"primary": oneHand, "held": oneHand}},
	{"shieldheld", "a shield and a held item",
		map[string]handUse{"shield": oneHand, "held": oneHand}},
	{"none", "nothing", nil},
}

func findHands(abbr string) (handChoice, bool) {
	for _, h := range handChoices {
		if h.abbr == abbr {
			return h, true
		}
	}
	return handChoice{}, false
}

// leaveOuts are the kinds of item a fill can be asked to leave out, by
// Kegor's load codes.
var leaveOuts = []struct{ abbr, display, load string }{
	{"rare", "rare items", "R"},
	{"quest", "quest rewards", "Q"},
	{"invasion", "invasion items", "I"},
}

// neverLoads are the load codes of items no fill uses: out of the game, and
// no take.
var neverLoads = []string{"!", "X"}

// bestOpts are what a fill is asked for, as the page and the build's link
// hold them: the stat, the stat after it if any, the way to fill the hands,
// and the abbrs of the leaveOuts to leave out, in their order.
type bestOpts struct {
	stat, then, hands string
	leave             []string
}

// defaultBest is what a fill is asked for unless the link says otherwise,
// and what the link leaves out.
var defaultBest = bestOpts{stat: "hit", hands: "any"}

// readBest reads what a fill is asked for from a form or link, putting the
// default in place of anything the page doesn't offer.
func readBest(v url.Values) bestOpts {
	o := bestOpts{stat: defaultBest.stat, hands: defaultBest.hands}
	if s, ok := findBestStat(v.Get("best")); ok {
		o.stat = s.abbr
	}
	if s, ok := findBestStat(v.Get("then")); ok && s.abbr != o.stat {
		o.then = s.abbr
	}
	if h, ok := findHands(v.Get("hands")); ok {
		o.hands = h.abbr
	}
	for _, l := range leaveOuts {
		if hasWord(v["leave"], l.abbr) {
			o.leave = append(o.leave, l.abbr)
		}
	}
	return o
}

// addValues adds what the fill is asked for to a build's link values: only
// what differs from defaultBest.
func (o bestOpts) addValues(v url.Values) {
	if o.stat != defaultBest.stat {
		v.Set("best", o.stat)
	}
	if o.then != "" {
		v.Set("then", o.then)
	}
	if o.hands != defaultBest.hands {
		v.Set("hands", o.hands)
	}
	for _, l := range o.leave {
		v.Add("leave", l)
	}
}

// score is what an item or a set gives a fill: the stat asked for, then the
// stat after it, then lightness, each counted so that more is better. One
// score beats another by the first of them that differs.
type score [3]int

func (a score) plus(b score) score {
	return score{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func (a score) less(b score) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// charParts are a character's parts, in buildChar's order, each with what
// the page offers for it.
var charParts = []struct {
	get    func(*buildChar) *string
	values func() []string
}{
	{func(c *buildChar) *string { return &c.class }, func() []string {
		var names []string
		for _, c := range buildClasses {
			names = append(names, c.name)
		}
		return names
	}},
	{func(c *buildChar) *string { return &c.race }, func() []string {
		var names []string
		for _, r := range buildRaces {
			names = append(names, r.name)
		}
		return names
	}},
	{func(c *buildChar) *string { return &c.align }, func() []string {
		var abbrs []string
		for _, a := range aligns {
			abbrs = append(abbrs, a.Abbr)
		}
		return abbrs
	}},
	{func(c *buildChar) *string { return &c.sex }, func() []string {
		var abbrs []string
		for _, s := range sexes {
			abbrs = append(abbrs, s.Abbr)
		}
		return abbrs
	}},
}

// users is a bit for each class, race, alignment and sex the page offers,
// by its place in charParts' values: who can use an item with no warning,
// or who a fill's items must suit.
type users [4]uint32

// usersOf finds who can use an item with no warning, as barred judges it.
func usersOf(it *buildItem) users {
	var u users
	for p, part := range charParts {
		for i, v := range part.values() {
			var c buildChar
			*part.get(&c) = v
			if len(c.barred(it)) == 0 {
				u[p] |= 1 << i
			}
		}
	}
	return u
}

// suitedBy is who a fill's items must suit for the character c: each part c
// leaves blank stands for everything the page offers for it. An illithid has
// no sex for an item to suit.
func suitedBy(c buildChar) users {
	var need users
	for p, part := range charParts {
		chosen := *part.get(&c)
		for i, v := range part.values() {
			if chosen == "" || chosen == v {
				need[p] |= 1 << i
			}
		}
	}
	if c.race == noSexRace {
		need[3] = 0
	}
	return need
}

// suits reports whether everyone in need can use the item u is for.
func (u users) suits(need users) bool {
	for p := range u {
		if u[p]&need[p] != need[p] {
			return false
		}
	}
	return true
}

// candidate is an item a fill can choose, with what it gives the fill.
type candidate struct {
	it    *buildItem
	score score
}

// wholeFlag is the flag of an item worn in the slot with key that covers
// other slots, or "" if none can.
func wholeFlag(key string) string {
	for _, c := range covers {
		if c.slot == key {
			return c.flag
		}
	}
	return ""
}

// poolKey names the pool of candidates a slot draws an item from: the kind
// of slot, with " whole" for an item on the head or body that covers other
// slots, and " 2h" for an item in a hand slot that takes both hands. The two
// slots of a pair, and the two weapons, share a pool.
func poolKey(s buildSlot, it *buildItem) string {
	if f := wholeFlag(s.key); f != "" && it.whole(f) {
		return s.slot + " whole"
	}
	if hasWord(handSlots, s.key) && it.twoHanded() {
		return s.slot + " 2h"
	}
	return s.slot
}

// candidates reads the items a fill can choose from for the character c:
// those that give the fill something, fit a slot, and that any player can
// have and c can use.
func candidates(db *sql.DB, c buildChar, o bestOpts) ([]*candidate, error) {
	st, _ := findBestStat(o.stat)
	then, hasThen := findBestStat(o.then)
	test := st.sql + " != 0"
	if hasThen {
		test = "(" + test + " OR " + then.sql + " != 0)"
	}
	var args []interface{}
	for _, l := range neverLoads {
		args = append(args, l)
	}
	for _, l := range leaveOuts {
		if hasWord(o.leave, l.abbr) {
			args = append(args, l.load)
		}
	}
	marks := strings.TrimSuffix(strings.Repeat("?, ", len(args)), ", ")
	noTake, targs := hasToken("item_flags", "NOTAKE")
	items, err := queryItems(db, " WHERE short_stats != '' AND "+test+
		" AND coalesce(load, '') NOT IN ("+marks+") AND NOT "+noTake,
		append(args, targs...)...)
	if err != nil {
		return nil, err
	}

	// many items share their flags, so who can use them is found once for
	// each set of flags
	need := suitedBy(c)
	suits := map[string]bool{}
	var cands []*candidate
	for _, it := range items {
		sc := score{st.dir * it.vals[st.key], 0, -it.weight}
		if hasThen {
			sc[1] = then.dir * it.vals[then.key]
		}
		if !(score{}).less(sc) || !slices.ContainsFunc(buildSlots,
			func(s buildSlot) bool { return s.fits(it) }) {
			continue
		}
		flags := strings.Join(it.flags, " ") + "|" + strings.Join(it.only, ",")
		ok, found := suits[flags]
		if !found {
			ok = usersOf(it).suits(need)
			suits[flags] = ok
		}
		if ok {
			cands = append(cands, &candidate{it, sc})
		}
	}
	return cands, nil
}

// filler finds the best set for a fill from its candidates, by pool, best
// first: around what a set keeps, if kept says it keeps the slot, with the
// item found there in keptItems, and with the tail only for a yuan-ti.
type filler struct {
	pools     map[string][]*candidate
	hands     handChoice
	tail      bool
	kept      []bool
	keptItems []*buildItem
}

func newFiller(cands []*candidate, o bestOpts, tail bool, kept []bool,
	keptItems []*buildItem) *filler {
	f := &filler{pools: map[string][]*candidate{}, tail: tail, kept: kept,
		keptItems: keptItems}
	f.hands, _ = findHands(o.hands)
	slices.SortFunc(cands, func(a, b *candidate) int {
		switch {
		case b.score.less(a.score):
			return -1
		case a.score.less(b.score):
			return 1
		}
		return strings.Compare(a.it.name, b.it.name)
	})
	for _, c := range cands {
		in := map[string]bool{}
		for _, s := range buildSlots {
			if k := poolKey(s, c.it); s.fits(c.it) && !in[k] {
				in[k] = true
				f.pools[k] = append(f.pools[k], c)
			}
		}
	}
	return f
}

// best is the best candidate in a pool, or nil.
func (f *filler) best(pool string) *candidate {
	if p := f.pools[pool]; len(p) > 0 {
		return p[0]
	}
	return nil
}

// plan is the best set for the slots the set doesn't keep: the candidate
// for each slot, by buildSlots index, or nil.
func (f *filler) plan() []*candidate {
	picks := make([]*candidate, len(buildSlots))
	put := func(key string) {
		if n := slotIndex(key); !f.kept[n] {
			picks[n] = f.best(buildSlots[n].slot)
		}
	}

	var mains, under []string
	for _, c := range covers {
		if !hasWord(mains, c.slot) {
			mains = append(mains, c.slot)
		}
		under = append(under, c.what)
	}
	for _, s := range buildSlots {
		switch {
		case hasWord(mains, s.key), hasWord(under, s.key),
			hasWord(handSlots, s.key):
		case s.key == "tail":
			if f.tail {
				put(s.key)
			}
		default:
			put(s.key)
		}
	}

	// a whole-body or whole-head item, or what it would cover
	for _, main := range mains {
		var covered []string
		for _, c := range covers {
			if c.slot == main {
				covered = append(covered, c.what)
			}
		}
		n := slotIndex(main)
		if f.kept[n] {
			if it := f.keptItems[n]; it == nil || !it.whole(wholeFlag(main)) {
				for _, k := range covered {
					put(k)
				}
			}
			continue
		}
		kind := buildSlots[n].slot
		parts, open := score{}, true
		for _, k := range append([]string{main}, covered...) {
			i := slotIndex(k)
			if k != main && f.kept[i] {
				open = false
			} else if c := f.best(buildSlots[i].slot); c != nil {
				parts = parts.plus(c.score)
			}
		}
		if whole := f.best(kind + " whole"); open && whole != nil &&
			parts.less(whole.score) {
			picks[n] = whole
			continue
		}
		picks[n] = f.best(kind)
		for _, k := range covered {
			put(k)
		}
	}

	f.fillHands(picks)
	return picks
}

// fillHands fills the hand slots the set doesn't keep, the best way the
// hands choice allows: two hands in all, counting what the set keeps, an
// item that takes both counting twice, and a secondary weapon only with a
// primary one.
func (f *filler) fillHands(picks []*candidate) {
	left := 2
	for _, k := range handSlots {
		if n := slotIndex(k); f.kept[n] {
			left--
			if it := f.keptItems[n]; it != nil && it.twoHanded() {
				left--
			}
		}
	}
	at := func(key string) int { return slices.Index(handSlots, key) }
	cur := make([]*candidate, len(handSlots))
	var top []*candidate
	var topScore score
	var try func(i, left int, sum score)
	try = func(i, left int, sum score) {
		if i == len(handSlots) {
			if cur[at("secondary")] != nil && cur[at("primary")] == nil &&
				!f.kept[slotIndex("primary")] {
				return
			}
			if top == nil || topScore.less(sum) {
				top, topScore = slices.Clone(cur), sum
			}
			return
		}
		try(i+1, left, sum)
		key := handSlots[i]
		n := slotIndex(key)
		if f.kept[n] {
			return
		}
		use := f.hands.use[key]
		for _, way := range []struct {
			pool  string
			hands int
			ok    bool
		}{
			{buildSlots[n].slot, 1, use == oneHand || use == anyHands},
			{buildSlots[n].slot + " 2h", 2, key != "secondary" &&
				(use == anyHands || use == bothHands)},
		} {
			if !way.ok || way.hands > left {
				continue
			}
			if c := f.best(way.pool); c != nil {
				cur[i] = c
				try(i+1, left-way.hands, sum.plus(c.score))
				cur[i] = nil
			}
		}
	}
	try(0, left, score{})
	for i, c := range top {
		picks[slotIndex(handSlots[i])] = c
	}
}

// bestSet finds the best set for a fill for the character c, around the
// items a set keeps: the candidate for each slot it fills, by buildSlots
// index, or nil.
func bestSet(db *sql.DB, c buildChar, o bestOpts, kept []bool,
	keptItems []*buildItem) ([]*candidate, error) {
	cands, err := candidates(db, c, o)
	if err != nil {
		return nil, err
	}
	return newFiller(cands, o, c.race == tailRace, kept, keptItems).plan(), nil
}

// fill fills a set with the best items for the fill the build asks for, as
// the page's Fill button asks: the set fillTo names, or the first empty one,
// in place of what it had, or around it if keep is checked. A set that gets
// anything is labeled for the stat, unless it keeps a label.
func (b *build) fill(db *sql.DB, v url.Values) error {
	n := b.target(v.Get("fillTo"))
	if n < 0 {
		return nil
	}
	set := buildSet{names: map[string]string{}}
	if v.Get("keep") != "" {
		set.label = b.sets[n].label
		maps.Copy(set.names, b.sets[n].names)
	}
	kept := make([]bool, len(buildSlots))
	keptItems := make([]*buildItem, len(buildSlots))
	find := newFinder(db)
	for i, s := range buildSlots {
		if name := set.names[s.key]; name != "" {
			it, err := find.slotItem(name, s)
			if err != nil {
				return err
			}
			kept[i], keptItems[i] = true, it
		}
	}
	picks, err := bestSet(db, b.char, b.best, kept, keptItems)
	if err != nil {
		return err
	}
	filled := false
	for i, c := range picks {
		if c != nil {
			set.names[buildSlots[i].key] = c.it.name
			filled = true
		}
	}
	if filled && set.label == "" {
		st, _ := findBestStat(b.best.stat)
		set.label = field("Best "+st.display, maxLabel)
	}
	b.sets[n] = set
	return nil
}
