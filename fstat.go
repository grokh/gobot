package main

import (
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// fstat finds items in Kegor's eq table by what they do. A search is fields
// separated by commas: a stat compared with a number (dam > 2), or a
// resist, slot, effect, flag, who can use an item, type, weapon type or
// zone. readFstat reads each field the way players write them: hp=10 as
// well as hp = 10, max_str or max strength for maxstr, ears for ear, and a
// slot, effect or flag named alone. A field it still can't read isn't
// skipped: the search doesn't run, and the reply names the field, with a
// guess at what was meant if there's one (guessField).

// fstatOps maps the comparison operators fstat accepts, as symbols or
// words, to SQL. The operator is written into the SQL text (it can't be a ?
// parameter), so it's only ever one of these values, never raw text from a
// tell.
var fstatOps = map[string]string{
	"=": "=", "==": "=", "eq": "=",
	"!=": "!=", "<>": "!=", "ne": "!=",
	"<": "<", "lt": "<",
	"<=": "<=", "=<": "<=", "le": "<=",
	">": ">", "gt": ">",
	">=": ">=", "=>": ">=", "ge": ">=",
}

// opWords are the words for fstat's operators, by their SQL, which its
// links write in place of the symbols.
var opWords = map[string]string{
	"=": "eq", "!=": "ne", "<": "lt", "<=": "le", ">": "gt", ">=": "ge",
}

// fstatLimit is the most items an fstat reply lists.
var fstatLimit = 10

// torileqURL is the TorilEQ site.
const torileqURL = "https://www.grokh.com/torileq/"

var (
	// a comparison with the operator as a symbol, with or without spaces
	// around it, and maybe a % after the number: dam > 2, hp=10, fire>5%
	fstatSymbolCmp = regexp.MustCompile(
		`^(.*?)\s*(>=|=>|<=|=<|!=|<>|==|=|>|<)\s*([-+]?\d{1,6})%?$`)
	// a comparison with the operator as a word: dam gt 2
	fstatWordCmp = regexp.MustCompile(
		`(?i)^(.*?)\s+(gt|ge|lt|le|eq|ne)\s+([-+]?\d{1,6})%?$`)
	// a word saying what the field looks for, then what, maybe after a :
	// or an =: slot neck, slot: neck
	fstatKindValue = regexp.MustCompile(`^([A-Za-z]+)\s*[:=]?\s*(.*)$`)
	// a name alone, as players write them: on_body, max str, !sleep
	fstatName = regexp.MustCompile(`^[A-Za-z!][A-Za-z _'!-]*$`)
	// a name of one word
	fstatWord = regexp.MustCompile(`^[A-Za-z!][A-Za-z_'!-]*$`)
)

// fstatKinds are the words that say what a field looks for, as players
// write them: slot ear, slots ear and worn ear are all slot ear.
var fstatKinds = func() map[string]string {
	kinds := make(map[string]string)
	for kind, words := range map[string][]string{
		"resist": {"resists", "resistance", "resistances"},
		"slot":   {"slots", "worn", "wear"},
		"effect": {"effects", "affect", "affects"},
		"flag":   {"flags"},
		"usable": {"useable"},
		"type":   {"types"},
		"weapon": {"weapons"},
		"zone":   {"zones"},
	} {
		kinds[kind] = kind
		for _, w := range words {
			kinds[w] = kind
		}
	}
	return kinds
}()

// fstatAliases are names players use that the tables in items.go don't
// have, and the things (by name) they mean.
var fstatAliases = map[string]string{
	// stats; armor, the Armor affect, is part of ac (acSQL)
	"damage": "dam", "hps": "hp", "armor": "ac",
	// slots
	"belt": "waist", "boots": "feet", "helm": "head", "helmet": "head",
	"ring": "finger", "sleeves": "arms",
	// who can use an item, as the TorilEQ form calls them
	"rogue": "thief", "neutral": "neut",
}

// fstatField is one field of an fstat search, read.
type fstatField struct {
	// stat, resist, slot, effect, flag, usable, type, weapon or zone
	kind string
	// the stat, resist, slot, effect, flag or who, as help fstat names it,
	// or the type, weapon type or zone as written
	name string
	// for a stat or resist compared with a number, the operator, as SQL
	op  string
	num int
	// its test in SQL, with the ? values it takes, and the ORDER BY it
	// asks for, if any
	test  string
	args  []interface{}
	order string
}

// text is the field as fstat writes it: with its operator as a word in
// links (dam gt 2), or as a symbol in a reply (dam > 2). A stat alone is its
// name, and anything else follows its kind: slot neck.
func (f fstatField) text(words bool) string {
	switch {
	case f.op != "":
		op := f.op
		if words {
			op = opWords[op]
		}
		text := f.name + " " + op + " " + strconv.Itoa(f.num)
		if f.kind == "resist" {
			text = "resist " + text
		}
		return text
	case f.kind == "stat":
		return f.name
	}
	return f.kind + " " + f.name
}

// fstatThing is something a field can name: a stat, resist, slot, effect,
// flag, or who can use an item. The tables in items.go, which the TorilEQ
// form searches on too, list them all.
type fstatThing struct {
	kind, name string
	sql        string // a stat's or resist's value, as SQL (see stat.sql)
	// the test for anything else, with the ? values it takes
	match func() (string, []interface{})
}

// field is the field for t named alone, or after its kind: a stat the item
// has any of, the most first, whether it's above 0 or below; a resist above
// 0; or the slot, effect, flag or who.
func (t fstatThing) field() fstatField {
	f := fstatField{kind: t.kind, name: t.name}
	switch t.kind {
	case "stat":
		f.test = t.sql + " != 0"
		f.order = "abs(" + t.sql + ") DESC"
	case "resist":
		f.test = t.sql + " > 0"
	default:
		f.test, f.args = t.match()
	}
	return f
}

// compare is the field comparing t, a stat or resist, with num. Items come
// by it, the most first, or the least for < and <=.
func (t fstatThing) compare(op string, num int) fstatField {
	f := fstatField{kind: t.kind, name: t.name, op: op, num: num,
		test: t.sql + " " + op + " ?", args: vals(num),
		order: t.sql + " DESC"}
	if op == "<" || op == "<=" {
		f.order = t.sql + " ASC"
	}
	return f
}

// fstatThings are what fields can name, and fstatNames every name each one
// goes by, as fstatKey writes it, with the places in fstatThings of the
// things that go by it.
var fstatThings, fstatNames = newFstatThings()

func newFstatThings() ([]fstatThing, map[string][]int) {
	var things []fstatThing
	names := make(map[string][]int)
	add := func(t fstatThing, aliases ...string) {
		n := len(things)
		things = append(things, t)
		for _, a := range append(aliases, t.name) {
			if k := fstatKey(a); k != "" && !slices.Contains(names[k], n) {
				names[k] = append(names[k], n)
			}
		}
	}
	for _, s := range stats {
		add(fstatThing{kind: "stat", name: strings.ToLower(s.abbr),
			sql: s.sql()}, s.col, s.display)
	}
	for _, r := range resists {
		var aliases []string
		for _, n := range []string{r.abbr, r.display} {
			aliases = append(aliases, n, "resist "+n, n+" resist")
		}
		add(fstatThing{kind: "resist", name: r.abbr, sql: col(r.col)},
			append(aliases, r.col)...)
	}
	for _, s := range slots {
		s := s // go.mod's Go 1.21 shares a loop's variable between its turns
		add(fstatThing{kind: "slot", name: s.abbr,
			match: func() (string, []interface{}) {
				return hasToken("worn", s.worn...)
			}}, append([]string{s.display}, s.worn...)...)
	}
	for _, e := range effects {
		e := e
		add(fstatThing{kind: "effect", name: e.abbr,
			match: func() (string, []interface{}) {
				return hasToken("affect_flags", e.token)
			}}, e.display, e.token)
	}
	for _, f := range flags {
		add(fstatThing{kind: "flag", name: f.abbr, match: f.match},
			append(append([]string{f.display}, f.flags...), f.worn...)...)
	}
	for _, s := range supps {
		s := s
		add(fstatThing{kind: "flag", name: s.abbr,
			match: func() (string, []interface{}) { return s.test, nil }},
			s.display)
	}
	for _, u := range usableBy {
		u := u
		add(fstatThing{kind: "usable", name: strings.TrimPrefix(u.abbr, "!"),
			match: func() (string, []interface{}) {
				test, args := hasToken("item_flags", u.restrict)
				return "NOT " + test, args
			}})
	}
	for alias, name := range fstatAliases {
		for n, t := range things {
			if t.name == name {
				names[alias] = append(names[alias], n)
			}
		}
	}
	return things, names
}

// fstatKey is a name the way fstat compares names: in lower case, with
// nothing but its letters and digits, so max_str, Max Str and MAXSTR are
// all maxstr.
func fstatKey(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// ofKind is the things at places in fstatThings that are of one of kinds,
// or of any kind if kinds is empty, each once, in fstatThings' order.
func ofKind(places []int, kinds []string) []fstatThing {
	places = slices.Clone(places) // it may be one of fstatNames' own
	slices.Sort(places)
	places = slices.Compact(places)
	var things []fstatThing
	for _, n := range places {
		if len(kinds) == 0 || slices.Contains(kinds, fstatThings[n].kind) {
			things = append(things, fstatThings[n])
		}
	}
	return things
}

// findThing is every thing of kinds (of any kind, if none are given) that
// name could mean, trying each of these in turn until one finds something:
//   - name as one of a thing's names
//   - name without a plural s (ears), or an "on" before it (on_wrist), or
//     with "no" for a ! before it (!sleep), leaving three letters or more
//   - the start of a thing's name, three letters or more (max agil)
//   - for a name of one word, a thing's name, four letters or more, with
//     more after it (sf_necro is sf_nec, maxagil maxagi)
//
// It's more than one thing if name could mean either.
func findThing(name string, kinds ...string) []fstatThing {
	key := fstatKey(name)
	if key == "" {
		return nil
	}
	if found := ofKind(fstatNames[key], kinds); len(found) > 0 {
		return found
	}
	var places []int
	for _, v := range []string{strings.TrimSuffix(key, "s"),
		strings.TrimPrefix(key, "on")} {
		// not dis for di, which would be a guess
		if v != key && len(v) >= 3 {
			places = append(places, fstatNames[v]...)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(name), "!") {
		places = append(places, fstatNames["no"+key]...)
	}
	if found := ofKind(places, kinds); len(found) > 0 {
		return found
	}
	if len(key) >= 3 {
		places = nil
		for n, at := range fstatNames {
			if strings.HasPrefix(n, key) {
				places = append(places, at...)
			}
		}
		if found := ofKind(places, kinds); len(found) > 0 {
			return found
		}
	}
	if !fstatWord.MatchString(strings.TrimSpace(name)) {
		return nil
	}
	// the longest name that starts name: a shorter one, like cha for
	// chainmail, would be a guess
	longest := 0
	places = nil
	for n, at := range fstatNames {
		if len(n) < 4 || len(n) < longest || !strings.HasPrefix(key, n) ||
			len(ofKind(at, kinds)) == 0 {
			continue
		}
		if len(n) > longest {
			longest, places = len(n), nil
		}
		places = append(places, at...)
	}
	return ofKind(places, kinds)
}

// closestThing is the thing of kinds (of any kind, if none are given) with
// the name fewest edits from name, if that's near enough to be a typo of it
// (see typoEdits) and no other thing has a name as near: max_wiz is maxwis.
func closestThing(name string, kinds ...string) (fstatThing, bool) {
	key := fstatKey(name)
	if len(key) < 3 {
		return fstatThing{}, false
	}
	most := typoEdits(key)
	best := most
	var places []int
	for n, at := range fstatNames {
		if len(n) < len(key)-most || len(n) > len(key)+most ||
			len(ofKind(at, kinds)) == 0 {
			continue
		}
		switch e := edits(key, n); {
		case e < best:
			best, places = e, append([]int(nil), at...)
		case e == best:
			places = append(places, at...)
		}
	}
	if found := ofKind(places, kinds); len(found) == 1 {
		return found[0], true
	}
	return fstatThing{}, false
}

// closestKind is the kind of field word is a typo of the word for, as
// slott is slot, or "" if it's near none.
func closestKind(word string) string {
	w := strings.ToLower(word)
	if len(w) < 3 {
		return ""
	}
	best, kind, tied := typoEdits(w), "", false
	for k, v := range fstatKinds {
		switch e := edits(w, k); {
		case e < best:
			best, kind, tied = e, v, false
		case e == best && kind == "":
			kind = v
		case e == best && v != kind:
			tied = true
		}
	}
	if tied {
		return ""
	}
	return kind
}

// readField reads one field of an fstat search. If it can't, could is what
// the field might mean when that's more than one thing, as evil is a resist
// and who can use an item.
func readField(text string) (f fstatField, could []fstatField, ok bool) {
	s := strings.Join(strings.Fields(text), " ")
	kind, value := "", ""
	if m := fstatKindValue.FindStringSubmatch(s); m != nil {
		kind, value = fstatKinds[strings.ToLower(m[1])], m[2]
	}
	switch {
	case kind == "type" || kind == "weapon" || kind == "zone":
		return readFree(kind, value)
	case fstatSymbolCmp.MatchString(s):
		m := fstatSymbolCmp.FindStringSubmatch(s)
		return readCompare(m[1], fstatOps[m[2]], m[3])
	case fstatWordCmp.MatchString(s):
		m := fstatWordCmp.FindStringSubmatch(s)
		return readCompare(m[1], fstatOps[strings.ToLower(m[2])], m[3])
	case kind != "" && value != "":
		found := findThing(value, kind)
		if len(found) == 0 && (kind == "flag" || kind == "effect") {
			// players mix them up: flag no_sum is effect no_sum
			found = findThing(value, "flag", "effect")
		}
		return fromThings(found)
	case kind == "" && fstatName.MatchString(s):
		return fromThings(findThing(s))
	}
	return fstatField{}, nil, false
}

// fromThings is the field naming the thing found, if one was, or what it
// might mean, if several were.
func fromThings(found []fstatThing) (fstatField, []fstatField, bool) {
	if len(found) == 1 {
		return found[0].field(), nil, true
	}
	var could []fstatField
	for _, t := range found {
		could = append(could, t.field())
	}
	return fstatField{}, could, false
}

// readFree reads a type, weapon or zone field, whose value fstat looks for
// as written, ignoring case: Kegor's names for them are many, and change.
func readFree(kind string, value string) (fstatField, []fstatField, bool) {
	if value == "" {
		return fstatField{}, nil, false
	}
	f := fstatField{kind: kind, name: value}
	switch kind {
	case "type":
		// LIQUID_CONT is liquid cont too
		f.test, f.args = squashed("type")+" = ?", vals(squash(value))
	case "weapon":
		// Short Sword is shortsword too, and Longsword long sword
		f.test, f.args = squashed("w_type")+" = ?", vals(squash(value))
	case "zone":
		// the zone, or one of the zones of an item that loads in several
		f.test = "instr(? || lower(zone) || ?, ?) > 0"
		f.args = vals(zoneSep, zoneSep,
			zoneSep+strings.ToLower(value)+zoneSep)
	}
	return f, nil, true
}

// compared is the stat or resist a comparison names, and the kinds it can
// be: a stat or a resist, or only a resist after the word resist.
func compared(what string) (string, []string) {
	if w := strings.SplitN(what, " ", 2); len(w) == 2 &&
		fstatKinds[strings.ToLower(w[0])] == "resist" {
		return w[1], []string{"resist"}
	}
	return what, []string{"stat", "resist"}
}

// readCompare reads a field comparing a stat or resist, called what, with
// num, by op, one of fstatOps' SQL operators.
func readCompare(what, op, num string) (fstatField, []fstatField, bool) {
	n, err := strconv.Atoi(num)
	if err != nil {
		return fstatField{}, nil, false
	}
	what, kinds := compared(what)
	found := findThing(what, kinds...)
	if len(found) == 1 {
		return found[0].compare(op, n), nil, true
	}
	var could []fstatField
	for _, t := range found {
		could = append(could, t.compare(op, n))
	}
	return fstatField{}, could, false
}

// guessLimitWords is the most words guessField splits into fields.
const guessLimitWords = 12

// guessField is what a field fstat can't read might have meant, if there's
// a fair guess, as one field or several:
//   - with a typo in the word saying what to look for: slott neck
//   - as fields with their commas left out: slot finger dam > 2
//   - with a typo in a name: max_wiz > 0
func guessField(text string) []fstatField {
	s := strings.Join(strings.Fields(text), " ")
	words := strings.Split(s, " ")
	if len(words) > 1 {
		if k := closestKind(words[0]); k != "" &&
			fstatKinds[strings.ToLower(words[0])] == "" {
			if f, _, ok := readField(k + " " + strings.Join(words[1:], " ")); ok {
				return []fstatField{f}
			}
		}
		if len(words) <= guessLimitWords {
			if fields := splitField(words); len(fields) > 1 {
				return fields
			}
		}
	}
	var kind, value string
	if m := fstatKindValue.FindStringSubmatch(s); m != nil {
		kind, value = fstatKinds[strings.ToLower(m[1])], m[2]
	}
	switch {
	case fstatSymbolCmp.MatchString(s) || fstatWordCmp.MatchString(s):
		m := fstatSymbolCmp.FindStringSubmatch(s)
		op := ""
		if m != nil {
			op = fstatOps[m[2]]
		} else {
			m = fstatWordCmp.FindStringSubmatch(s)
			op = fstatOps[strings.ToLower(m[2])]
		}
		n, err := strconv.Atoi(m[3])
		what, kinds := compared(m[1])
		if t, ok := closestThing(what, kinds...); ok && err == nil {
			return []fstatField{t.compare(op, n)}
		}
	case kind == "slot" || kind == "effect" || kind == "flag" ||
		kind == "usable" || kind == "resist":
		kinds := []string{kind}
		if kind == "flag" || kind == "effect" {
			kinds = []string{"flag", "effect"}
		}
		if t, ok := closestThing(value, kinds...); ok {
			return []fstatField{t.field()}
		}
	case kind == "" && fstatName.MatchString(s):
		if t, ok := closestThing(s); ok {
			return []fstatField{t.field()}
		}
	}
	return nil
}

// splitField reads words as fields with the commas between them left out,
// fewest fields first, each read as it stands. It's nil if they can't be.
func splitField(words []string) []fstatField {
	for i := 1; i < len(words); i++ {
		f, _, ok := readField(strings.Join(words[:i], " "))
		if !ok {
			continue
		}
		if g, _, ok := readField(strings.Join(words[i:], " ")); ok {
			return []fstatField{f, g}
		}
		if more := splitField(words[i:]); more != nil {
			return append([]fstatField{f}, more...)
		}
	}
	return nil
}

// fstatPart is one field of an fstat search, as written and as read.
type fstatPart struct {
	text  string
	ok    bool       // whether it could be read
	field fstatField // what it was read as
	// if it couldn't be read: the things it could mean, if it could mean
	// several, or else a guess at what it meant, as one field or several
	could []fstatField
	guess []fstatField
}

// fstatQuery is an fstat search, read.
type fstatQuery []fstatPart

// readFstat reads an fstat search: fields separated by commas, with or
// without spaces. An empty one is left out.
func readFstat(oper string) fstatQuery {
	var q fstatQuery
	for _, text := range strings.Split(oper, ",") {
		if text = strings.TrimSpace(text); text == "" {
			continue
		}
		p := fstatPart{text: text}
		p.field, p.could, p.ok = readField(text)
		if !p.ok && len(p.could) == 0 {
			p.guess = guessField(text)
		}
		q = append(q, p)
	}
	return q
}

// unread is the fields q couldn't read.
func (q fstatQuery) unread() []fstatPart {
	var parts []fstatPart
	for _, p := range q {
		if !p.ok {
			parts = append(parts, p)
		}
	}
	return parts
}

// sql is the tests after WHERE for the fields q read, with the values they
// take, and its ORDER BY: by the first field that asks for an order, then
// by name.
func (q fstatQuery) sql() (where string, args []interface{}, order string) {
	var tests []string
	for _, p := range q {
		if !p.ok {
			continue
		}
		tests = append(tests, p.field.test)
		args = append(args, p.field.args...)
		if order == "" {
			order = p.field.order
		}
	}
	if order != "" {
		order += ", "
	}
	return strings.Join(tests, " AND "), args, order + "name"
}

// text is the fields q read as fstat writes them, with operators as words
// for links, or as symbols for a reply: dam gt 2, slot neck.
func (q fstatQuery) text(words bool) string {
	var fields []string
	for _, p := range q {
		if p.ok {
			fields = append(fields, p.field.text(words))
		}
	}
	return strings.Join(fields, ", ")
}

// fixed is q with each field it couldn't read replaced by the guess at what
// it meant, if every one of them has one.
func (q fstatQuery) fixed() (fstatQuery, bool) {
	var f fstatQuery
	for _, p := range q {
		if p.ok {
			f = append(f, p)
			continue
		}
		if len(p.guess) == 0 {
			return nil, false
		}
		for _, g := range p.guess {
			f = append(f, fstatPart{ok: true, field: g})
		}
	}
	return f, true
}

// cantRead is the line saying which of q's fields fstat couldn't read.
func (q fstatQuery) cantRead() string {
	var fields []string
	for _, p := range q.unread() {
		fields = append(fields, "'"+p.text+"'")
	}
	return "fstat can't read " + orList(fields) + "."
}

// couldLimit is the most things a reply offers for a field that could mean
// several.
const couldLimit = 8

// hint is the line after cantRead in the reply to the fstat search oper,
// read as q: the search as it might have been meant, if every field fstat
// couldn't read has a guess; what the one field it couldn't read could be;
// or stat, if oper is an item's name, as now and then it is. Failing those,
// it points to help fstat.
func (q fstatQuery) hint(oper string) string {
	if f, ok := q.fixed(); ok {
		return "Did you mean 'tell katumi fstat " + f.text(false) + "'?"
	}
	unread := q.unread()
	if could := unread[0].could; len(unread) == 1 && len(could) > 1 &&
		len(could) <= couldLimit {
		var texts []string
		for _, f := range could {
			texts = append(texts, f.text(false))
		}
		return "Did you mean " + orList(texts) + "?"
	}
	if len(q) == 1 && namesItem(oper) {
		return "Did you mean 'tell katumi stat " + strings.TrimSpace(oper) +
			"'?"
	}
	return "See 'tell katumi help fstat'."
}

// namesItem reports whether stat would find an item called name, with
// each of name's words a whole word of the item's name: a lyre, but not a
// slotted ring for slot.
func namesItem(name string) bool {
	db, err := OpenItemDB()
	if err != nil {
		return false
	}
	defer db.Close()
	for _, m := range nameMatches(name) {
		items, err := findStats(db, m, "short_stats")
		if err != nil || len(items) == 0 {
			continue
		}
		words := strings.FieldsFunc(strings.ToLower(items[0].name),
			func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
					r != '\'' && r != '-'
			})
		for _, w := range strings.Fields(strings.ToLower(name)) {
			if !slices.Contains(words, w) {
				return false
			}
		}
		return true
	}
	return false
}

// Fstat lists up to fstatLimit items from TorilDB.db matching every field
// in oper, as readFstat reads them, and says how many more there are. A
// search with a field fstat can't read isn't run: the reply says which, and
// what might have been meant.
func Fstat(oper string) []string {
	switch strings.ToLower(strings.TrimSpace(oper)) {
	case "?", "help":
		return Help("fstat")
	}
	q := readFstat(oper)
	if len(q.unread()) > 0 {
		log.Printf("fstat can't read: %s\n", oper)
		return []string{q.cantRead(), q.hint(oper)}
	}
	if len(q) == 0 {
		return []string{BadSyntax}
	}

	db, err := OpenItemDB()
	if err != nil {
		log.Printf("%s: %v\n", itemDBFile, err)
		return []string{itemsDown}
	}
	defer db.Close()
	where, args, order := q.sql()
	txts, more, err := queryStats(db, "SELECT short_stats FROM eq WHERE "+
		where+" ORDER BY "+order, fstatLimit, args...)
	if err != nil {
		log.Printf("%s: fstat %s: %v\n", itemDBFile, oper, err)
		return []string{itemsDown}
	}
	if len(txts) == 0 {
		return []string{NotFound("item(s)", oper)}
	}
	for n, txt := range txts {
		txts[n] = tintinSafe(txt)
	}
	if more > 0 {
		txts = append(txts, fstatMore(q, more))
	}
	return txts
}

// fstatMore is the line ending an fstat reply that leaves items out: how
// many, and where TorilEQ lists them all, if the search fits in a link.
func fstatMore(q fstatQuery, more int) string {
	line := "...and " + strconv.Itoa(more) + " more"
	if link := fstatLink(q); link != "" {
		line += ": " + link
	}
	return line
}

// fstatLink is the TorilEQ link to everything the search q finds, written
// as fstat writes it, with its operators as words, as in
// ?fstat=dam+gt+2,+slot+neck. It's "" if the search has anything but
// letters, digits, spaces and ,.'!_- in it, so the link never needs escaping
// and stays whole through tintin, the MUD and the clients that make links
// of addresses, if it would be too long for one tell, or if q has a field
// fstat couldn't read.
func fstatLink(q fstatQuery) string {
	if len(q) == 0 || len(q.unread()) > 0 {
		return ""
	}
	query := strings.ReplaceAll(q.text(true), " ", "+")
	for _, c := range query {
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' ||
			'0' <= c && c <= '9' || strings.ContainsRune(",.'!_-+", c)) {
			return ""
		}
	}
	link := torileqURL + "?fstat=" + query
	if len(link) > tellLimit {
		return ""
	}
	return link
}

// squash is a name in lower case without spaces or underscores, for fstat
// to match with squashed.
func squash(name string) string {
	return strings.NewReplacer(" ", "", "_", "").Replace(strings.ToLower(name))
}

// squashed is the SQL for an eq column's text as squash leaves a name.
func squashed(column string) string {
	return "replace(replace(lower(" + col(column) + "), ' ', ''), '_', '')"
}
