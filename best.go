package main

import (
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
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
// character the page leaves blank stands for every class, alignment or sex
// the page offers, and a race left blank for every race but yuan-ti, by the
// owner's choice, so a set suits them all (see standsFor): the designer
// can't tell which of them the game lets go together, so it never picks
// one. Players wear two of the same ring, earring or bracer, as their logs
// show, so a pair's two slots can hold the same item.
//
// Asked for the essential resists, a fill makes the set the best one with at
// least 1% of each of them, counting what it keeps, which can mean an item
// worn for its resist alone. That's the set it would make without them if
// that set has them. If no set has them all, the set is the best one with
// as many of them as any has.
//
// A cap on the stat counts it only up to the cap, or for a stat that's better
// less, only down to it, so that what's left goes to the second stat: the set
// is the best for the second stat of those with as much of the first as the
// cap, if any set has that much, and as many of the essentials as any. With
// no second stat, it's the lightest.

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

// essentials are the resists a set needs at least 1% of each of to keep
// dragon breath from destroying its equipment, by their abbrs in resists.
var essentials = [...]string{"fire", "cold", "acid"}

// sums are what a set, or an item, gives of what a plan can count: the stat
// a fill is asked for, as score counts it, at capDim, and essentials[i] at
// 1+i.
type sums [1 + len(essentials)]int

// capDim is where sums hold the stat a fill is asked for.
const capDim = 0

// essentialDims are where sums hold the essentials.
func essentialDims() []int {
	var dims []int
	for i := range essentials {
		dims = append(dims, 1+i)
	}
	return dims
}

// maxCap is the most a cap may be, either way from 0.
const maxCap = 9999

// andList joins words as a sentence lists them: a, b and c.
func andList(words []string) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " +
		words[len(words)-1]
}

// bestOpts are what a fill is asked for, as the page and the build's link
// hold them: the stat, the stat after it if any, the way to fill the hands,
// the abbrs of the leaveOuts to leave out, in their order, whether the set
// needs the essentials, and if capped, the most of the stat that counts,
// which for a stat that's better less is the least.
type bestOpts struct {
	stat, then, hands string
	leave             []string
	resists, capped   bool
	capAt             int
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
	o.resists = v.Get("resists") != ""
	if n, err := strconv.Atoi(strings.TrimSpace(v.Get("cap"))); err == nil &&
		n >= -maxCap && n <= maxCap {
		o.capped, o.capAt = true, n
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
	if o.resists {
		v.Set("resists", "1")
	}
	if o.capped {
		v.Set("cap", strconv.Itoa(o.capAt))
	}
}

// score is what an item or a set gives a fill: the stat asked for, then the
// stat after it, then lightness, each counted so that more is better. One
// score beats another by the first of them that differs. A set's stats run
// to a few thousand at most, so 32 bits hold them, and keep a plan's tables
// small (see way).
type score [3]int32

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

// standsFor is what part p of the character stands for with chosen as its
// value: that value, or if it's blank, everything the page offers for it,
// but for the race, every race but yuan-ti, whose body differs (see
// bareSlots), as the owner chose.
func standsFor(p int, chosen string) []string {
	if chosen != "" {
		return []string{chosen}
	}
	var c buildChar
	race := charParts[p].get(&c) == &c.race
	var values []string
	for _, v := range charParts[p].values() {
		if !race || v != tailRace {
			values = append(values, v)
		}
	}
	return values
}

// suitedBy is who a fill's items must suit for the character c: what each
// of its parts stands for. An illithid has no sex for an item to suit.
func suitedBy(c buildChar) users {
	var need users
	for p, part := range charParts {
		stands := standsFor(p, *part.get(&c))
		for i, v := range part.values() {
			if hasWord(stands, v) {
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

// candidate is an item a fill can choose, with what it gives the fill: its
// score, and its sums, which hold the essentials only if the fill needs
// them.
type candidate struct {
	it    *buildItem
	score score
	res   sums
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
// those that give the fill something, or give one of the essentials if it
// needs them, that fit a slot, and that any player can have and c can use.
func candidates(db *sql.DB, c buildChar, o bestOpts) ([]*candidate, error) {
	st, _ := findBestStat(o.stat)
	then, hasThen := findBestStat(o.then)
	tests := []string{st.sql + " != 0"}
	if hasThen {
		tests = append(tests, then.sql+" != 0")
	}
	if o.resists {
		for _, e := range essentials {
			r, _ := findBestStat(e)
			tests = append(tests, r.sql+" > 0")
		}
	}
	test := "(" + strings.Join(tests, " OR ") + ")"
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
		sc := score{int32(st.dir * it.vals[st.key]), 0, int32(-it.weight)}
		if hasThen {
			sc[1] = int32(then.dir * it.vals[then.key])
		}
		res := sums{capDim: int(sc[0])}
		gives := (score{}).less(sc)
		if o.resists {
			for i, e := range essentials {
				res[1+i] = it.vals["r:"+e]
				gives = gives || res[1+i] > 0
			}
		}
		if !gives || !slices.ContainsFunc(buildSlots,
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
			cands = append(cands, &candidate{it, sc, res})
		}
	}
	return cands, nil
}

// filler finds the best set for a fill from its candidates, by pool, best
// first: around what a set keeps, if kept says it keeps the slot, with the
// item found there in keptItems, with nothing in the slots bare names (see
// bareSlots) nor a whole item that would cover one, with the essentials if
// need says so, and counting no more of the stat than its cap, if capped.
//
// It plans the set a slot at a time, keeping in a table the best way it has
// found to fill the slots so far. A plan that counts no sums keeps one way,
// so it's the best item for each slot, or for a group of slots, such as a
// whole-body item against what it covers. One that counts sums, of the
// essentials or of a capped stat, keeps the best way for each sums a way
// gives, so that it can end on the best way that reaches each sum's goal:
// 1% of an essential, or the cap. It counts a sum only as far as tells
// whether the set ends on its goal: no further down than the most the slots
// it has yet to fill can take away, and no further up than enough to stay on
// the goal after they take that much.
type filler struct {
	pools     map[string][]*candidate
	hands     handChoice
	bare      []string
	kept      []bool
	keptItems []*buildItem
	need      bool
	stat      bestStat // the stat asked for
	capped    bool
	capAt     int // its cap, as score counts the stat

	// the plan under way: the sums it counts, by index in sums, whether it
	// counts the capped stat apart from the score (see value), the goal of
	// each sum, the least and most of each it counts a way as giving, the
	// sums a table holds at each index, each pool's front, and the tables
	// it's done with, which it uses again (see newTable)
	on     []int
	apart  bool
	goal   sums
	lo, hi sums
	at     []sums
	fronts map[string][]*candidate
	spare  []table
}

// newTable is an empty table for the plan under way: one it's done with, or
// a new one. A capped plan's table can take megabytes, and making a new one
// for each step would have a fill take several times the memory it needs.
func (f *filler) newTable() table {
	if n := len(f.spare); n > 0 {
		t := f.spare[n-1]
		f.spare = f.spare[:n-1]
		clear(t)
		return t
	}
	return make(table, len(f.at))
}

// done gives back a table the plan under way no longer needs.
func (f *filler) done(t table) {
	f.spare = append(f.spare, t)
}

// maxWays is the most sums a plan may count, a table of 6 MB, which keeps a
// fill to about 90 MB on a server with 1 GB. With TorilDB.db's items in
// November 2026, no plan for the essentials, for a class or a race, counts
// more than 528, and the biggest capped plan tried, a cleric's hit points
// capped at 700 with the essentials, counts 201,888 and takes 70 MB.
var maxWays = 1 << 18

// table holds the best way a plan has found to fill the slots so far for
// each sums a way can give, at its index.
type table []way

// way is a way to fill the slots so far, if ok: its score, and the last of
// its picks. In this order its fields take 24 bytes.
type way struct {
	score score
	ok    bool
	last  *pick
}

// pick is a candidate a way puts in a slot, by buildSlots index, and the
// pick it made before.
type pick struct {
	n    int
	c    *candidate
	prev *pick
}

// merge puts each of u's ways in t where t has none for its sums, or one that
// scores less.
func (t table) merge(u table) {
	for i, w := range u {
		if w.ok && (!t[i].ok || t[i].score.less(w.score)) {
			t[i] = w
		}
	}
}

func newFiller(cands []*candidate, o bestOpts, race string, kept []bool,
	keptItems []*buildItem) *filler {
	st, _ := findBestStat(o.stat)
	f := &filler{pools: map[string][]*candidate{}, bare: bareSlots(race),
		kept: kept, keptItems: keptItems, need: o.resists, stat: st,
		capped: o.capped, capAt: st.dir * o.capAt}
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

// sumsOf is what an item gives of what a plan can count.
func (f *filler) sumsOf(it *buildItem) sums {
	s := sums{capDim: f.stat.dir * it.vals[f.stat.key]}
	for i, e := range essentials {
		s[1+i] = it.vals["r:"+e]
	}
	return s
}

// value is what a candidate gives the plan under way: its score, or when the
// plan counts the capped stat apart, the rest of its score.
func (f *filler) value(c *candidate) score {
	if f.apart {
		return score{c.score[1], c.score[2], 0}
	}
	return c.score
}

// front is the candidates in a pool the plan under way chooses from, best
// first by their value: each one but those valued no more than one before
// them that gives as much of each sum the plan counts. Without sums, it's
// the best candidate alone.
func (f *filler) front(pool string) []*candidate {
	if fr, ok := f.fronts[pool]; ok {
		return fr
	}
	cands := f.pools[pool]
	if f.apart {
		cands = slices.Clone(cands)
		slices.SortStableFunc(cands, func(a, b *candidate) int {
			switch va, vb := f.value(a), f.value(b); {
			case vb.less(va):
				return -1
			case va.less(vb):
				return 1
			}
			return 0
		})
	}
	var fr []*candidate
	for _, c := range cands {
		if !slices.ContainsFunc(fr, func(b *candidate) bool {
			for _, d := range f.on {
				if b.res[d] < c.res[d] {
					return false
				}
			}
			return true
		}) {
			fr = append(fr, c)
		}
	}
	f.fronts[pool] = fr
	return fr
}

// prepare readies a plan that counts the sums on lists, apart from the score
// if apart says to, with a table as big as the sums it may count, and gives
// the sums it starts with: what the set's kept items give.
func (f *filler) prepare(on []int, apart bool) (sums, error) {
	f.on, f.apart, f.fronts, f.spare = on, apart, map[string][]*candidate{}, nil
	f.goal = sums{capDim: f.capAt}
	for _, d := range essentialDims() {
		f.goal[d] = 1
	}
	var start, lose sums
	for n, s := range buildSlots {
		if f.kept[n] {
			if it := f.keptItems[n]; it != nil {
				give := f.sumsOf(it)
				for _, d := range f.on {
					start[d] += give[d]
				}
			}
			continue
		}
		if hasWord(f.bare, s.key) ||
			hasWord(handSlots, s.key) && f.hands.use[s.key] == noHand {
			continue
		}
		// the most the slot can take away of each
		var most sums
		pools := []string{s.slot, s.slot + " 2h"}
		if !coversBare(f.bare, s.key) {
			pools = append(pools, s.slot+" whole")
		}
		for _, p := range pools {
			for _, c := range f.front(p) {
				for _, d := range f.on {
					most[d] = min(most[d], c.res[d])
				}
			}
		}
		for _, d := range f.on {
			lose[d] += most[d]
		}
	}
	size := 1
	for _, d := range f.on {
		f.hi[d] = f.goal[d] - lose[d]
		start[d] = min(start[d], f.hi[d])
		f.lo[d] = start[d] + lose[d]
		size *= f.hi[d] - f.lo[d] + 1
	}
	if size > maxWays {
		return start, fmt.Errorf("%w: %d, from %v to %v", errTooBig, size, f.lo,
			f.hi)
	}
	f.at = make([]sums, size)
	for i := range f.at {
		rest := i
		for k := len(f.on) - 1; k >= 0; k-- {
			d := f.on[k]
			width := f.hi[d] - f.lo[d] + 1
			f.at[i][d] = f.lo[d] + rest%width
			rest /= width
		}
	}
	return start, nil
}

// index is where a table holds the way that gives s.
func (f *filler) index(s sums) int {
	i := 0
	for _, d := range f.on {
		if s[d] < f.lo[d] || s[d] > f.hi[d] {
			panic(fmt.Sprintf("a fill counted %v, not from %v to %v", s,
				f.lo, f.hi))
		}
		i = i*(f.hi[d]-f.lo[d]+1) + s[d] - f.lo[d]
	}
	return i
}

// step is t after putting one of opts in slot n, or nothing if empty allows
// it. Where two ways tie, it keeps the first it finds: it starts from the
// ways that gave the most of the first sum it counts (the capped stat, then
// fire, cold and acid), and so on, and tries nothing before an item, and the
// items in opts' order.
func (f *filler) step(t table, n int, opts []*candidate, empty bool) table {
	out := f.newTable()
	for i := len(t) - 1; i >= 0; i-- {
		w := t[i]
		if !w.ok {
			continue
		}
		if empty && (!out[i].ok || out[i].score.less(w.score)) {
			out[i] = w
		}
		for _, c := range opts {
			s := f.at[i]
			for _, d := range f.on {
				s[d] = min(f.hi[d], s[d]+c.res[d])
			}
			j, sc := f.index(s), w.score.plus(f.value(c))
			if !out[j].ok || out[j].score.less(sc) {
				out[j] = way{score: sc, ok: true, last: &pick{n, c, w.last}}
			}
		}
	}
	return out
}

// slot is t after filling the slot with key, unless the set keeps it or the
// slot is bare. It's done with t, which may be what it gives back.
func (f *filler) slot(t table, key string) table {
	if n := slotIndex(key); !f.kept[n] && !hasWord(f.bare, key) {
		out := f.step(t, n, f.front(buildSlots[n].slot), true)
		f.done(t)
		return out
	}
	return t
}

// errTooBig is the error of a plan that would count more sums than maxWays.
var errTooBig = errors.New("a fill would count too many sums")

// filled is what a fill makes: the candidate for each slot it fills, by
// buildSlots index, or nil, the essentials the set lacks if the fill needs
// them, and whether it left out the cap, which it does when counting it
// with the essentials would take more sums than maxWays.
type filled struct {
	picks  []*candidate
	lacks  []string
	capOut bool
}

// plan is the best set for the slots the set doesn't keep. If the best set
// lacks any of the essentials the fill needs, it's the best set with them
// all, or the best of those with the most of them. If the best set has more
// of a capped stat than its cap, it's the best set, for what else it gives,
// of those with as much as the cap and as many of the essentials.
func (f *filler) plan() (filled, error) {
	var out filled
	picks, _, err := f.planWith(nil, false)
	if err != nil {
		return out, err
	}
	if f.need && len(f.lacks(picks)) > 0 {
		if picks, _, err = f.planWith(essentialDims(), false); err != nil {
			return out, err
		}
	}
	if f.capped && f.total(picks) > f.capAt {
		on := []int{capDim}
		if f.need {
			on = append(on, essentialDims()...)
		}
		capped, ok, err := f.planWith(on, true)
		switch {
		case errors.Is(err, errTooBig):
			out.capOut = true
		case err != nil:
			return out, err
		case ok:
			picks = capped
		}
	}
	out.picks = picks
	if f.need {
		out.lacks = f.lacks(picks)
	}
	return out, nil
}

// total is how much of the stat asked for a set with picks, as well as what
// it keeps, gives, as score counts it.
func (f *filler) total(picks []*candidate) int {
	n := 0
	for i, it := range f.keptItems {
		if f.kept[i] && it != nil {
			n += f.sumsOf(it)[capDim]
		}
	}
	for _, c := range picks {
		if c != nil {
			n += int(c.score[0])
		}
	}
	return n
}

// lacks is the essentials a set with picks, as well as what it keeps, has
// less than 1% of.
func (f *filler) lacks(picks []*candidate) []string {
	var got sums
	add := func(it *buildItem) {
		give := f.sumsOf(it)
		for _, d := range essentialDims() {
			got[d] += give[d]
		}
	}
	for n, it := range f.keptItems {
		if f.kept[n] && it != nil {
			add(it)
		}
	}
	for _, c := range picks {
		if c != nil {
			add(c.it)
		}
	}
	var lacks []string
	for i, e := range essentials {
		if got[1+i] < 1 {
			lacks = append(lacks, e)
		}
	}
	return lacks
}

// end is the way a plan ends on, and whether there's one: of the ways with
// as much of a capped stat as its cap, if the plan counts it, and of those
// with the most of the essentials it counts, the one that scores best.
func (f *filler) end(t table) (way, bool) {
	var top way
	most := -1
	capped := slices.Contains(f.on, capDim)
	for i, w := range t {
		if !w.ok || capped && f.at[i][capDim] < f.goal[capDim] {
			continue
		}
		has := 0
		for _, d := range f.on {
			if d != capDim && f.at[i][d] >= f.goal[d] {
				has++
			}
		}
		if has > most || has == most && top.score.less(w.score) {
			top, most = w, has
		}
	}
	return top, most >= 0
}

// planWith plans the best set, counting the sums on lists, apart from the
// score if apart says to, and reports whether a way ended as the plan needs.
func (f *filler) planWith(on []int, apart bool) ([]*candidate, bool, error) {
	start, err := f.prepare(on, apart)
	if err != nil {
		return nil, false, err
	}
	t := f.newTable()
	t[f.index(start)] = way{ok: true}

	var mains, under []string
	for _, c := range covers {
		if !hasWord(mains, c.slot) {
			mains = append(mains, c.slot)
		}
		under = append(under, c.what)
	}
	for _, s := range buildSlots {
		if !hasWord(mains, s.key) && !hasWord(under, s.key) &&
			!hasWord(handSlots, s.key) {
			t = f.slot(t, s.key)
		}
	}

	// a whole-body or whole-head item, unless it would cover a bare slot, or
	// what it would cover, which wins if they tie
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
					t = f.slot(t, k)
				}
			}
			continue
		}
		kind := buildSlots[n].slot
		parts, open := f.step(t, n, f.front(kind), true), true
		for _, k := range covered {
			open = open && !f.kept[slotIndex(k)]
			parts = f.slot(parts, k)
		}
		if open && !coversBare(f.bare, main) {
			whole := f.step(t, n, f.front(kind+" whole"), false)
			parts.merge(whole)
			f.done(whole)
		}
		f.done(t)
		t = parts
	}

	w, ok := f.end(f.fillHands(t))
	picks := make([]*candidate, len(buildSlots))
	for p := w.last; p != nil; p = p.prev {
		picks[p.n] = p.c
	}
	return picks, ok, nil
}

// fillHands is t after filling the hand slots the set doesn't keep, each way
// the hands choice allows: two hands in all, counting what the set keeps, an
// item that takes both counting twice, and a secondary weapon only with a
// primary one. Where two ways tie, the one tried first wins: leaving a slot
// empty first, then an item that takes one hand, then one that takes both,
// slot by slot. It's done with t.
func (f *filler) fillHands(t table) table {
	left := 2
	for _, k := range handSlots {
		if n := slotIndex(k); f.kept[n] {
			left--
			if it := f.keptItems[n]; it != nil && it.twoHanded() {
				left--
			}
		}
	}
	var out table
	var try func(i, left int, t table, primary, secondary bool)
	try = func(i, left int, t table, primary, secondary bool) {
		if i == len(handSlots) {
			switch {
			case secondary && !primary:
			case out == nil:
				out = f.newTable()
				copy(out, t)
			default:
				out.merge(t)
			}
			return
		}
		try(i+1, left, t, primary, secondary)
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
			if opts := f.front(way.pool); way.ok && way.hands <= left &&
				len(opts) > 0 {
				next := f.step(t, n, opts, false)
				try(i+1, left-way.hands, next,
					primary || key == "primary", secondary || key == "secondary")
				f.done(next)
			}
		}
	}
	try(0, left, t, f.kept[slotIndex("primary")], false)
	f.done(t)
	return out
}

// bestSet finds the best set for a fill for the character c, around the
// items a set keeps.
func bestSet(db *sql.DB, c buildChar, o bestOpts, kept []bool,
	keptItems []*buildItem) (filled, error) {
	cands, err := candidates(db, c, o)
	if err != nil {
		return filled{}, err
	}
	return newFiller(cands, o, c.race, kept, keptItems).plan()
}

// fill fills a set with the best items for the fill the build asks for, as
// the page's Fill button asks: the set fillTo names, or the first empty one,
// in place of what it had, or around it if keep is checked. A set that gets
// anything is labeled for the stat, and its cap if it has one, unless it
// keeps a label. The build notes any of the essentials the set lacks, if the
// fill needs them, and a cap the fill left out.
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
	made, err := bestSet(db, b.char, b.best, kept, keptItems)
	if err != nil {
		return err
	}
	b.lacks, b.capOut = made.lacks, made.capOut
	gotAny := false
	for i, c := range made.picks {
		if c != nil {
			set.names[buildSlots[i].key] = c.it.name
			gotAny = true
		}
	}
	if gotAny && set.label == "" {
		st, _ := findBestStat(b.best.stat)
		label := "Best " + st.display
		if b.best.capped && !made.capOut {
			label = st.display + " to " + strconv.Itoa(b.best.capAt)
		}
		set.label = field(label, maxLabel)
	}
	b.sets[n] = set
	return nil
}
