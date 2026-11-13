package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func NotFound(four string, oper string) string {
	log.Printf("404: %s: %s\n", four, oper)
	return "404 " + four + " not found: " + oper
}

// nameMatch is one way of matching the requested item name: eq's name
// column has to match every arg using op.
type nameMatch struct {
	op    string
	args  []string
	limit int
}

// nameMatches lists the ways FindItem matches a name, best match first.
func nameMatches(oper string) []nameMatch {
	var words []string
	for _, word := range strings.Fields(oper) {
		words = append(words, "%"+word+"%")
	}
	return []nameMatch{
		// exact item name
		{"=", []string{oper}, 12},
		// LIKE, anywhere in the name
		{"LIKE", []string{"%" + oper + "%"}, 1},
		// LIKE with %'s in place of spaces
		{"LIKE", []string{strings.Replace(" "+oper+" ", " ", "%", -1)}, 1},
		// every word, in any order
		{"LIKE", words, 1},
	}
}

// foundItem is an item a name match found: its name in eq, and its stats
// readied for the MUD.
type foundItem struct{ name, stats string }

// findStats runs one name match against eq, returning the items found, with
// their stats in the length asked for (short_stats or long_stats).
func findStats(db *sql.DB, m nameMatch, length string) ([]foundItem, error) {
	if len(m.args) == 0 {
		return nil, nil
	}
	query := "SELECT name, " + length + " FROM eq WHERE "
	args := make([]interface{}, len(m.args))
	for n, arg := range m.args {
		query += "name " + m.op + " ? AND "
		args[n] = arg
	}
	// a row without stats isn't an answer; of loose matches, the shortest
	// name is the one closest to what was asked for
	query += length + " != '' ORDER BY length(name), name LIMIT " +
		strconv.Itoa(m.limit)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []foundItem
	for rows.Next() {
		var name, text sql.NullString
		if err := rows.Scan(&name, &text); err != nil {
			return nil, err
		}
		res = append(res, foundItem{name.String, tintinSafe(text.String)})
	}
	return res, rows.Err()
}

// queryStats runs a query for one column of stats text, returning the first
// limit rows, or every row if limit is 0, and how many more there were.
func queryStats(db *sql.DB, query string, limit int, args ...interface{}) (
	[]string, int, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var res []string
	more := 0
	for rows.Next() {
		if limit > 0 && len(res) == limit {
			more++
			continue
		}
		var text sql.NullString
		if err := rows.Scan(&text); err != nil {
			return nil, 0, err
		}
		res = append(res, text.String)
	}
	return res, more, rows.Err()
}

// tintinChars are the characters tintin++ treats specially. katumi.tin
// runs every line gobot prints as a tintin command, and drops a line with
// a double quote in it at its #if check, so that becomes a single quote,
// as shellsafe does for the shell. The rest are neutralized because a
// reply can carry text gobot didn't write: a player's own words, or
// TorilDB.db, which is someone else's file, refreshed daily.
var tintinChars = strings.NewReplacer(
	`"`, `'`,
	`;`, `,`,
	`{`, `(`,
	`}`, `)`,
	`\`, ``,
	`$`, ``,
)

// tellSafe readies a line for katumi.tin. It also turns control characters
// into spaces: a newline would end the tell, and katumi.tin would run
// whatever followed it as a command of its own. It leaves @ alone, since
// replies show accounts as @name: @ only calls a tintin function when a {
// follows the name, and tellSafe lets no { through.
func tellSafe(s string) string {
	return strings.Map(func(r rune) rune {
		if r < ' ' || r == 0x7f {
			return ' '
		}
		return r
	}, tintinChars.Replace(s))
}

// tintinSafe readies TorilDB.db text for katumi.tin: tellSafe, with any @
// taken out as well.
func tintinSafe(s string) string {
	return tellSafe(strings.ReplaceAll(s, "@", ""))
}

// FindItem returns stats for the item named in oper from TorilDB.db, in
// the length asked for (short_stats or long_stats). Each way of matching the
// name is tried in turn, from exact to loosest.
func FindItem(oper string, length string) []string {
	return findItem(oper, length, false)
}

// Astat is astat's reply: the long stats for the item named in oper, each
// followed by a line naming the mobs Kegor lists as carrying it, if any.
func Astat(oper string) []string {
	return findItem(oper, "long_stats", true)
}

// findItem is FindItem, adding astat's Loads on lines if loads is set.
func findItem(oper string, length string, loads bool) []string {
	db, err := OpenItemDB()
	if err != nil {
		log.Printf("%s: %v\n", itemDBFile, err)
		return []string{itemsDown}
	}
	defer db.Close()

	for _, m := range nameMatches(oper) {
		items, err := findStats(db, m, length)
		if err != nil {
			// not a database, no eq table, a column gone: say so, rather
			// than that the item doesn't exist
			log.Printf("%s: %s: %v\n", itemDBFile, oper, err)
			return []string{itemsDown}
		}
		var res []string
		for _, it := range items {
			res = append(res, it.stats)
			if !loads {
				continue
			}
			if line := loadsOn(db, it.name); line != "" {
				res = append(res, line)
			}
		}
		if len(res) > 0 {
			return res
		}
	}
	return []string{NotFound("item", oper)}
}

func Help(oper string) []string {
	oper = strings.ToLower(oper)
	var txt []string
	switch oper {
	case "?":
		txt = append(txt, "Syntax: tell katumi ? -- "+
			"Katumi provides a full listing of valid commands.")
	case "help":
		txt = append(txt, "Syntax: tell katumi help <cmd> -- "+
			"Example: tell katumi help stat -- "+
			"Katumi explains the command named. "+
			"For a list of every command: tell katumi ?")
	case "hidden", "hidden?":
		txt = append(txt, "Syntax: tell katumi hidden -- "+
			"Katumi sends a tell in reply "+
			"if she can see you. If you receive no reply, you are hidden. "+
			"Katumi has permanent detect invis to ensure that won't cause "+
			"issues.")
	case "who":
		txt = append(txt, "Syntax: tell katumi who <acct/char> -- "+
			"Example: tell katumi who rynshana -- "+
			"Katumi provides the account name along with a list of every "+
			"known alt of the named character as a reply. Also works with "+
			"account names.")
	case "char":
		txt = append(txt, "Syntax: tell katumi char <char> -- "+
			"Example: tell katumi char rynshana -- "+
			"Katumi provides the account name along with full information "+
			"on the named character as a reply, "+
			"to include level, class, race, and date/time last seen.")
	case "find":
		txt = append(txt, "Syntax: tell katumi find <acct/char> -- "+
			"Example: tell katumi find rynshana -- "+
			"Katumi provides the account name along with the last known "+
			"sighting of any of that character's alts. "+
			"If they have an alt online, "+
			"the time will measure in seconds. "+
			"Also works with account names.")
	case "clist":
		txt = append(txt, "Syntax: tell katumi clist <acct/char> -- "+
			"Example: tell katumi clist rynshana -- "+
			"Katumi provides a full "+
			"listing of every known alt belonging to <char>, including race, "+
			"class, level, and date/time last seen, matching the format of "+
			"the 'char' command. Also works with account names.")
	case "class":
		txt = append(txt, "Syntax: tell katumi class <class> -- "+
			"Example: tell katumi class enchanter -- "+
			"Katumi provides a "+
			"list of alts matching the named class for characters who "+
			"are currently online, letting group leaders find useful "+
			"alts from the 'who' list.")
	case "name":
		txt = append(txt, "Syntax: tell katumi name <acct/char> -- "+
			"Example: tell katumi name rynshana -- "+
			"Katumi provides the real name the account's player shared "+
			"with 'addname', if they've shared one. Also works with "+
			"account names.")
	case "addname":
		txt = append(txt, "Syntax: tell katumi addname <name> -- "+
			"Example: tell katumi addname Pat -- "+
			"Katumi records <name> as the real name for your account, "+
			"which anyone can then see with 'name'. Use addname again "+
			"to change it.")
	case "delalt":
		txt = append(txt, "Syntax: tell katumi delalt <char> -- "+
			"Example: tell katumi delalt rynshana -- "+
			"Katumi no longer "+
			"provides information on the alt, removing it from 'clist', "+
			"'who', and 'find' commands. Only works for characters "+
			"attached to the same account requesting the removal.")
	case "addalt":
		txt = append(txt, "Syntax: tell katumi addalt <char> -- "+
			"Example: tell katumi addalt rynshana -- "+
			"Katumi begins "+
			"providing information on the named alt, who had previously "+
			"been removed with 'delalt', adding the character back to "+
			"'clist', 'who', and 'find' commands. Only works for chars "+
			"attached to the same account.")
	case "lr":
		txt = append(txt, "Syntax: tell katumi lr -- "+
			"Katumi provides a list of load "+
			"reports for the current boot. This could be rares or quests "+
			"other players have found or completed. Use the 'lrdel' command "+
			"to remove bad or out of date reports.")
		txt = append(txt, "Syntax: tell katumi lr <report> -- "+
			"Example: tell katumi lr timestop gnome at ako village -- "+
			"Katumi adds <report> "+
			"to the list of load reports for the current boot. If you find "+
			"a rare, global load, or complete a quest or the like, report "+
			"it along with a location so other players will know!")
	case "lrdel":
		txt = append(txt, "Syntax: tell katumi lrdel <num> -- "+
			"Example: tell katumi lrdel 3 -- "+
			"Katumi removes the "+
			"numbered item from the load reports, if a quest is completed "+
			"or a rare killed, or a report found to be inaccurate. Please "+
			"do not abuse this command - this service helps everyone.")
	case "stat":
		txt = append(txt, "Syntax: tell katumi stat <item> -- "+
			"Example: tell katumi stat isha cloak -- "+
			"Katumi provides stat info for the item named. "+
			"Use 'astat' for the long form, with keywords and flags. "+
			"The name search is fairly forgiving. Stats come from "+
			"Kegor's TorilDB, updated daily. Mwrite new stats to Kegor.")
	case "astat":
		txt = append(txt, "Syntax: tell katumi astat <item> -- "+
			"Example: tell katumi astat destruction sword -- "+
			"Katumi provides full "+
			"stat information for the item named, then the mobs known to "+
			"carry it, up to 5. Use 'stat' for short "+
			"text. The name search is fairly forgiving.")
	case "mstat":
		txt = append(txt, "Syntax: tell katumi mstat <mob> -- "+
			"Example: tell katumi mstat pit fiend -- "+
			"Katumi provides stat info for the mob named: level, race, size, "+
			"zone and flags, then the eq it's known to carry and the items "+
			"its quest uses. Mobs that share a name, like the bartender in "+
			"many zones, are listed together, up to 10. The name search is "+
			"fairly forgiving. Stats come from Kegor's TorilDB, updated "+
			"daily.")
	case "fstat":
		txt = append(txt, "Syntax: tell katumi fstat <field>[, <field>...] -- "+
			"Example: tell katumi fstat maxagi > 0, resist fire, slot ear -- "+
			"Lists up to 10 items matching every field, the most of the "+
			"first stat first (the least, for < and <=), then a TorilEQ "+
			"link to the rest. Separate fields "+
			"with a comma and a space. A field is <stat> <op> <num>, where "+
			"<op> is > < = >= <= != or gt lt eq ge le ne, or one of: "+
			"resist <resist> (above "+
			"0), slot <slot>, effect <effect>, flag <flag>, usable <who> "+
			"(leaves out items flagged against them), type <type>, "+
			"weapon <weapon type>, zone <zone>. Type, weapon and zone take "+
			"Kegor's names: type worn, weapon short sword, zone jotunheim.")
		var names []string
		for _, s := range stats {
			names = append(names, s.abbr)
		}
		txt = append(txt, "Stats: "+fstatList(names))
		names = nil
		for _, r := range resists {
			names = append(names, r.abbr)
		}
		txt = append(txt, "Resists: "+fstatList(names))
		names = nil
		for _, s := range slots {
			names = append(names, s.abbr)
		}
		txt = append(txt, "Slots: "+fstatList(names))
		names = nil
		for _, e := range effects {
			names = append(names, e.abbr)
		}
		txt = append(txt, "Effects: "+fstatList(names))
		names = nil
		for _, f := range flags {
			names = append(names, f.abbr)
		}
		for _, s := range supps {
			names = append(names, s.abbr)
		}
		txt = append(txt, "Flags: "+fstatList(names))
		names = nil
		for _, u := range usableBy {
			names = append(names, strings.TrimPrefix(u.abbr, "!"))
		}
		txt = append(txt, "Usable: "+fstatList(names))
	default:
		txt = append(txt, NotFound("help file", oper))
		if guess := guessCommand(oper); guess != "" {
			txt = append(txt, "Did you mean 'tell katumi help "+guess+"'?")
		}
	}
	return txt
}

// fstatList is the sorted, comma-separated list of one kind of field fstat
// accepts, taken from the table that defines them, so help fstat can't fall
// behind.
func fstatList(names []string) string {
	var list []string
	for _, n := range names {
		list = append(list, strings.ToLower(n))
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}

// fstatOps maps the comparison operators fstat accepts, as symbols or
// words, to SQL. The operator is written into the SQL text (it can't be a ?
// parameter), so it's only ever one of these values, never raw text from a
// tell.
var fstatOps = map[string]string{
	"=": "=", "==": "=", "eq": "=",
	"!=": "!=", "<>": "!=", "ne": "!=",
	"<": "<", "lt": "<",
	"<=": "<=", "le": "<=",
	">": ">", "gt": ">",
	">=": ">=", "ge": ">=",
}

// opWords are the words for fstat's operators, by their SQL, which
// fstatLink writes in place of the symbols.
var opWords = map[string]string{
	"=": "eq", "!=": "ne", "<": "lt", "<=": "le", ">": "gt", ">=": "ge",
}

// fstatLimit is the most items an fstat reply lists.
var fstatLimit = 10

// Fstat lists up to fstatLimit items from TorilDB.db matching every field
// in oper, as fstatSearch reads them, and says how many more there are.
func Fstat(oper string) []string {
	where, args, order, ok := fstatSearch(oper)
	switch {
	case !ok:
		return []string{NotFound("item(s)", oper)}
	case where == "":
		return []string{BadSyntax}
	}

	db, err := OpenItemDB()
	if err != nil {
		log.Printf("%s: %v\n", itemDBFile, err)
		return []string{itemsDown}
	}
	defer db.Close()
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
		txts = append(txts, fstatMore(oper, more))
	}
	return txts
}

// fstatSearch reads oper's fields, separated by a comma and a space, into
// fstat's search: the tests after WHERE, with their arguments, and the
// ORDER BY. Items come by the first stat compared, the most of it first,
// or the least for < and <=, then by name. ok is false if a field names a
// stat, resist, slot, effect, flag or who fstat doesn't know. A field in no
// form it knows is skipped.
func fstatSearch(oper string) (where string, args []interface{},
	order string, ok bool) {
	var tests []string
	ok = true
	for _, field := range strings.Split(oper, ", ") {
		test, targs, by, known := fstatField(strings.Fields(field))
		switch {
		case !known:
			ok = false
		case test != "":
			tests = append(tests, test)
			args = append(args, targs...)
			if order == "" {
				order = by
			}
		}
	}
	if order != "" {
		order += ", "
	}
	return strings.Join(tests, " AND "), args, order + "name", ok
}

// torileqURL is the TorilEQ site.
const torileqURL = "https://www.grokh.com/torileq/"

// fstatMore is the line ending an fstat reply that leaves items out: how
// many, and where TorilEQ lists them all, if oper fits in a link.
func fstatMore(oper string, more int) string {
	line := "...and " + strconv.Itoa(more) + " more"
	if link := fstatLink(oper); link != "" {
		line += ": " + link
	}
	return line
}

// fstatLink is the TorilEQ link to everything the fstat search in oper
// finds, with its operators as words, as in ?fstat=dam+gt+2,+slot+neck. It's
// "" if oper has anything but letters, digits, spaces and ,.'!_- in it, so
// the link never needs escaping and stays whole through tintin, the MUD and
// the clients that make links of addresses, and if it would be too long for
// one tell.
func fstatLink(oper string) string {
	var fields []string
	for _, field := range strings.Split(oper, ", ") {
		f := strings.Fields(field)
		if len(f) == 3 {
			if op, ok := fstatOps[strings.ToLower(f[1])]; ok {
				f[1] = opWords[op]
			}
		}
		fields = append(fields, strings.Join(f, "+"))
	}
	query := strings.Join(fields, ",+")
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

// fstatField is the SQL test for one fstat field, split into words, with
// its arguments. A field is a word saying what to look for, then what:
//   - resist <resist>: above 0
//   - slot <slot>
//   - effect <effect>: an ability the item gives
//   - flag <flag>: one of the TorilEQ form's item flags, or where an item
//     comes from (is_rare, for_quest and so on)
//   - usable <who>: leaves out items flagged against a class group, sex,
//     alignment or race, as the form's Usable By does. A few class-only
//     items, like the HELLBORNE-ONLY ones, have no such flags, so it lets
//     them through.
//   - type <type>, weapon <weapon type> or zone <zone>, by Kegor's names
//
// or "<stat> <op> <num>", for which order is the ORDER BY that puts the
// best of that stat first. ok is false for a field naming a stat, resist,
// slot, effect, flag or who fstat doesn't know. test is "" for a field in
// no form fstat knows.
func fstatField(f []string) (test string, args []interface{}, order string,
	ok bool) {
	if len(f) < 2 {
		return "", nil, "", true
	}
	what := strings.Join(f[1:], " ")
	switch strings.ToLower(f[0]) {
	case "resist":
		if r, ok := findResist(what); ok {
			return col(r.col) + " > 0", nil, "", true
		}
	case "slot":
		if s, ok := findSlot(what); ok {
			test, args := hasToken("worn", s.worn...)
			return test, args, "", true
		}
	case "effect":
		for _, e := range effects {
			if strings.EqualFold(e.abbr, what) {
				test, args := hasToken("affect_flags", e.token)
				return test, args, "", true
			}
		}
	case "flag":
		for _, fl := range flags {
			if strings.EqualFold(fl.abbr, what) {
				test, args := fl.match()
				return test, args, "", true
			}
		}
		for _, s := range supps {
			if strings.EqualFold(s.abbr, what) {
				return s.test, nil, "", true
			}
		}
	case "usable":
		for _, u := range usableBy {
			if strings.EqualFold(u.abbr, "!"+strings.TrimPrefix(what, "!")) {
				test, args := hasToken("item_flags", u.restrict)
				return "NOT " + test, args, "", true
			}
		}
	case "type":
		// LIQUID_CONT is liquid cont too
		return squashed("type") + " = ?", vals(squash(what)), "", true
	case "weapon":
		// Short Sword is shortsword too, and Longsword long sword
		return squashed("w_type") + " = ?", vals(squash(what)), "", true
	case "zone":
		// the zone, or one of the zones of an item that loads in several
		return "instr(? || lower(zone) || ?, ?) > 0",
			vals(zoneSep, zoneSep, zoneSep+strings.ToLower(what)+zoneSep), "",
			true
	default:
		op, isOp := fstatOps[strings.ToLower(f[1])]
		if len(f) != 3 || !isOp {
			return "", nil, "", true
		}
		s, ok := findStat(f[0])
		if !ok {
			return "", nil, "", false
		}
		order = col(s.col) + " DESC" // the most first
		if op == "<" || op == "<=" {
			order = col(s.col) + " ASC" // the least
		}
		return col(s.col) + " " + op + " ?", vals(f[2]), order, true
	}
	return "", nil, "", false
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

func Who(oper string) string {
	db := OpenDB()
	defer db.Close()

	var txt string
	query := "SELECT account_name, char_name " +
		"FROM chars WHERE vis = 't' " +
		"AND (account_name = " +
		"(SELECT account_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
		"OR LOWER(account_name) = LOWER(?)) " +
		"ORDER BY char_level DESC, char_name ASC"

	rows, err := db.Query(query, oper, oper)
	ChkErr(err)
	defer rows.Close()

	var c Char
	for rows.Next() {
		ChkErr(rows.Scan(&c.acct, &c.name))
		if strings.Contains(txt, "@") {
			txt += ", " + c.name
		} else {
			txt = "@" + c.acct
			txt += ": " + c.name
		}
	}
	ChkRows(rows)

	if !strings.Contains(txt, "@") {
		txt = NotFound("character or account", oper)
	}
	return txt
}

func Clist(oper string) []string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT char_level, class_name, char_name, char_race, " +
		"account_name, DATETIME(last_seen) " +
		"FROM chars WHERE vis = 't' " +
		"AND (account_name = " +
		"(SELECT account_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
		"OR LOWER(account_name) = LOWER(?)) " +
		"ORDER BY char_level DESC, char_name ASC"

	rows, err := db.Query(query, oper, oper)
	ChkErr(err)
	defer rows.Close()

	var txts []string
	var found bool
	var c Char
	for rows.Next() {
		ChkErr(rows.Scan(&c.lvl, &c.class, &c.name,
			&c.race, &c.acct, &c.seen))
		char := fmt.Sprintf(
			"[%d %s] %s (%s) (@%s) seen %s UTC",
			c.lvl, c.class, c.name, c.race, c.acct, c.seen,
		)
		txts = append(txts, char)
		found = true
	}
	ChkRows(rows)
	if !found {
		txts = append(txts, NotFound("character or account", oper))
	}
	return txts
}

func CharInfo(oper string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT char_level, class_name, char_name, char_race, " +
		"account_name, DATETIME(last_seen) " +
		"FROM chars WHERE vis = 't' " +
		"AND LOWER(char_name) = LOWER(?)"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var c Char
	err = stmt.QueryRow(oper).Scan(
		&c.lvl, &c.class, &c.name, &c.race, &c.acct, &c.seen)
	if err == sql.ErrNoRows {
		txt = NotFound("character", oper)
	} else if err != nil {
		log.Fatal(err)
	} else {
		txt = fmt.Sprintf(
			"[%d %s] %s (%s) (@%s) seen %s UTC",
			c.lvl, c.class, c.name, c.race, c.acct, c.seen,
		)
	}
	return txt
}

func Find(oper string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT account_name, char_name, " +
		"(STRFTIME('%s','now') - STRFTIME('%s',last_seen)) seconds " +
		"FROM chars WHERE vis = 't' " +
		"AND (account_name = " +
		"(SELECT account_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
		"OR LOWER(account_name) = LOWER(?)) " +
		"ORDER BY JULIANDAY(last_seen) DESC LIMIT 1"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var seconds int
	var c Char
	err = stmt.QueryRow(oper, oper).Scan(&c.acct, &c.name, &seconds)
	if err == sql.ErrNoRows {
		txt = NotFound("character or account", oper)
	} else if err != nil {
		log.Fatal(err)
	} else {
		online := false
		secs := time.Duration(seconds) * time.Second
		if secs.Hours() >= 24 && secs.Hours() < 48 {
			c.seen = "1 day"
		} else if secs.Hours() >= 48 {
			days := int(secs.Hours()) / 24
			c.seen = fmt.Sprintf("%d days", days)
		} else if secs.Seconds() > 3600 {
			hours := int(secs.Seconds()) / 3600
			minutes := (int(secs.Seconds()) % 3600) / 60
			c.seen = fmt.Sprintf("%dh%dm", hours, minutes)
		} else if secs.Seconds() > 60 {
			minutes := int(secs.Seconds()) / 60
			seconds = int(secs.Seconds()) % 60
			c.seen = fmt.Sprintf("%dm%ds", minutes, seconds)
		} else if secs.Seconds() <= 60 && secs.Seconds() >= 0 {
			c.seen = fmt.Sprintf("%ds", int(secs.Seconds()))
			online = true
		} else {
			log.Printf("'find' error: seconds were %d\n", int(secs.Seconds()))
		}
		// Char.seen = secs.String() // easier :/
		if !online {
			txt = fmt.Sprintf(
				"@%s last seen %s ago as %s",
				c.acct, c.seen, c.name)
		} else {
			txt = fmt.Sprintf(
				"@%s is online as %s",
				c.acct, c.name)
		}
	}
	return txt
}

func Name(oper string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT account_name, player_name " +
		"FROM accounts " +
		"WHERE player_name IS NOT NULL " +
		"AND (account_name = " +
		"(SELECT account_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
		"OR LOWER(account_name) = LOWER(?))"

	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var c Char
	err = stmt.QueryRow(oper, oper).Scan(&c.acct, &c.name)
	if err == sql.ErrNoRows {
		query := "SELECT account_name " +
			"FROM accounts " +
			"WHERE (account_name = " +
			"(SELECT account_name FROM chars " +
			"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
			"OR LOWER(account_name) = LOWER(?))"

		stmt, err := db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		err = stmt.QueryRow(oper, oper).Scan(&c.acct)
		if err == sql.ErrNoRows {
			txt = NotFound("character or account", oper)
		} else if err != nil {
			log.Fatal(err)
		} else {
			txt = fmt.Sprintf("@%s did not disclose their real name", c.acct)
		}
	} else if err != nil {
		log.Fatal(err)
	} else {
		txt = fmt.Sprintf("@%s's real name is %s", c.acct, c.name)
	}
	return txt
}

func FindClass(oper string) []string {
	db := OpenDB()
	defer db.Close()

	// stored times carry different offsets (or none), so compare them as
	// times, not as text: as text, the hour after clocks fall back sorts
	// ahead of the hour before it
	query := "SELECT char_name, class_name, char_race, " +
		"char_level, account_name " +
		"FROM chars WHERE LOWER(class_name) = LOWER(?) " +
		"AND vis = 't' " +
		"AND account_name IN " +
		"(SELECT account_name FROM chars " +
		"WHERE JULIANDAY(last_seen) > JULIANDAY('now', '-1 minute') " +
		"AND vis = 't') " +
		"ORDER BY char_level DESC"

	rows, err := db.Query(query, oper)
	ChkErr(err)
	defer rows.Close()

	var txts []string
	var found bool
	var c Char
	for rows.Next() {
		ChkErr(rows.Scan(&c.name, &c.class, &c.race,
			&c.lvl, &c.acct))
		txt := fmt.Sprintf(
			"[%d %s] %s (%s) (@%s)",
			c.lvl, c.class, c.name, c.race, c.acct,
		)
		txts = append(txts, txt)
		found = true
	}
	ChkRows(rows)
	if !found {
		txts = append(txts, NotFound("class", oper))
	}
	return txts
}

func DelAlt(oper string, char string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT account_name, char_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) " +
		"AND vis = 't' AND account_name = " +
		"(SELECT account_name FROM chars WHERE char_name = ?)"

	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var c Char
	err = stmt.QueryRow(oper, char).Scan(&c.acct, &c.name)
	if err == sql.ErrNoRows {
		txt = NotFound("character or account", oper)
	} else if err != nil {
		log.Fatal(err)
	} else {
		query = "UPDATE chars SET vis = 'f' " +
			"WHERE LOWER(char_name) = LOWER(?)"

		tx, err := db.Begin()
		ChkErr(err)
		stmt, err := tx.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		_, err = stmt.Exec(oper)
		ChkErr(err)
		tx.Commit()
		log.Printf("Delalt: %s\n", oper)
		txt = fmt.Sprintf("Removed character from your alt list:: %s", oper)
	}
	return txt
}

func AddAlt(oper string, char string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT account_name, char_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) " +
		"AND account_name = " +
		"(SELECT account_name FROM chars WHERE char_name = ?)"

	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var c Char
	err = stmt.QueryRow(oper, char).Scan(&c.acct, &c.name)
	if err == sql.ErrNoRows {
		txt = NotFound("character or account", oper)
	} else if err != nil {
		log.Fatal(err)
	} else {
		query = "UPDATE chars SET vis = 't' " +
			"WHERE LOWER(char_name) = LOWER(?)"

		tx, err := db.Begin()
		ChkErr(err)
		stmt, err := tx.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		_, err = stmt.Exec(oper)
		ChkErr(err)
		tx.Commit()
		txt = fmt.Sprintf("Re-added character to your alt list:: %s", oper)
	}
	return txt
}

// titleCase gives each word of a name a capital first letter and leaves the
// rest as typed, so "mary-jane o'brien" becomes "Mary-Jane O'Brien". It does
// what strings.Title did, which Go has deprecated: a word starts wherever
// the character before isn't part of one.
func titleCase(s string) string {
	startsWord := true
	return strings.Map(func(r rune) rune {
		first := startsWord
		startsWord = !inWord(r)
		if first {
			return unicode.ToTitle(r)
		}
		return r
	}, s)
}

// inWord reports whether r is part of a word for titleCase: a letter, a
// digit or an underscore, and, as in strings.Title, anything outside ASCII
// that isn't a space.
func inWord(r rune) bool {
	if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	return r >= utf8.RuneSelf && !unicode.IsSpace(r)
}

func AddName(oper string, char string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT account_name " +
		"FROM chars " +
		"WHERE char_name = ?"

	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var c Char
	err = stmt.QueryRow(char).Scan(&c.acct)
	if err == sql.ErrNoRows {
		txt = NotFound("character", char)
	} else if err != nil {
		log.Fatal(err)
	} else {
		query = "UPDATE accounts " +
			"SET player_name = ? " +
			"WHERE account_name = ?"

		tx, err := db.Begin()
		ChkErr(err)
		stmt, err := tx.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		name := titleCase(oper)
		_, err = stmt.Exec(name, c.acct)
		ChkErr(err)
		tx.Commit()
		txt = fmt.Sprintf("Your real name recorded as: %s", oper)
	}
	return txt
}

func LoadReport(oper string, char string) []string {
	db := OpenDB()
	defer db.Close()

	var txts []string
	var txt string
	var c Char
	if oper == "" {
		query := "SELECT char_name, report_text, " +
			"DATETIME(report_time) " +
			"FROM loads WHERE boot_id = " +
			"(SELECT MAX(boot_id) FROM boots) " +
			"AND deleted = 'f' ORDER BY rowid"

		rows, err := db.Query(query)
		ChkErr(err)
		defer rows.Close()

		var found bool
		counter := 1
		for rows.Next() {
			var report string
			var date string
			ChkErr(rows.Scan(&c.name, &report, &date))
			txt = fmt.Sprintf(
				"%d: %s [%s at %s UTC]",
				counter, report, c.name, date,
			)
			txts = append(txts, txt)
			counter++
			found = true
		}
		ChkRows(rows)
		if !found {
			txts = append(txts, "No loads reported for current boot.")
		}
	} else if strings.Contains(oper, " ") {
		tx, err := db.Begin()
		ChkErr(err)

		date := now().UTC()
		query := "INSERT INTO loads " +
			"VALUES((SELECT MAX(boot_id) FROM boots), ?, ?, ?, 'f')"

		stmt, err := tx.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		_, err = stmt.Exec(date, oper, char)
		ChkErr(err)
		tx.Commit()
		txts = append(txts, fmt.Sprintf("Load reported: %s", oper))
	} else {
		txts = append(txts, BadSyntax)
	}
	return txts
}

func LRDel(oper string) string {
	db := OpenDB()
	defer db.Close()

	num, err := strconv.Atoi(oper)
	if err != nil {
		return BadSyntax
	}

	// the same reports, in the same order, that lr lists
	query := "SELECT rowid, report_text FROM loads " +
		"WHERE deleted = 'f' " +
		"AND boot_id = (SELECT MAX(boot_id) FROM boots) ORDER BY rowid"

	rows, err := db.Query(query)
	ChkErr(err)
	defer rows.Close()

	var txt string
	var row int64
	report := ""
	counter := 1
	for rows.Next() {
		var id int64
		var text string
		ChkErr(rows.Scan(&id, &text))
		if counter == num {
			row = id
			report = text
		}
		counter++
	}
	ChkRows(rows)
	if counter > 1 {
		if report != "" {
			// by rowid: matching on report_time needed the stored time read
			// back in exactly the form it was written
			_, err = db.Exec("UPDATE loads SET deleted = 't' WHERE rowid = ?",
				row)
			ChkErr(err)
			txt = fmt.Sprintf("Load deleted: %s", report)
		} else {
			txt = "Invalid load report number."
		}
	} else {
		txt = "No loads reported for current boot."
	}
	return txt
}

var BadSyntax string = "Invalid syntax. For valid syntax: tell katumi ?, " +
	"tell katumi help <cmd>"

func ReplyTo(char string, tell string) []string {
	info := []string{"I am a Helper Bot (Beta). Each command I accept " +
		"has further help files available at: tell katumi help <cmd>",
		"Find items: Acronymed stats: stat <item name>, " +
			"Stats fully spelled out: astat <item name>, " +
			"Find items by attributes, slots, etc.: (for proper usage, " +
			"tell katumi help fstat)",
		"Find mobs: Stats and the eq they carry: mstat <mob name>",
		"Find people: Provide acct and char info: who <char/acct>, " +
			"clist <char/acct>, char <char>, Show last online alt: " +
			"find <char/acct>, Find alts of listed class for people online: " +
			"class <class>, RL names: name <char/acct>, addname <name>, " +
			"Control your listing: delalt <char>, addalt <char>",
		"Misc: This message: ?, More info on each command: help <cmd>, " +
			"Find out if you're hidden: hidden, Load reports for rares or " +
			"global mobs: lr, lr <report>, lrdel <num>",
	}
	var txt []string

	oper := ""
	split := strings.SplitN(tell, " ", 2)
	cmd := strings.ToLower(split[0])
	if len(split) > 1 {
		oper = strings.TrimSpace(split[1])
	}

	// debugging
	//log.Printf("Cmd : %s\n", cmd)
	//log.Printf("Oper: %s\n", oper)

	switch {
	case cmd == "?":
		txt = info
	case cmd == "help" && oper == "":
		txt = info
	case cmd == "help" && oper != "":
		txt = Help(oper)
	case strings.HasPrefix(cmd, "hidden"):
		if char != "Someone" {
			txt = append(txt, char+" is NOT hidden!")
		}
	case cmd == "stat" && oper != "":
		txt = withGuess(FindItem(oper, "short_stats"), "eq", oper)
	case cmd == "astat" && oper != "":
		txt = withGuess(Astat(oper), "eq", oper)
	case cmd == "mstat" && oper != "":
		txt = withGuess(FindMob(oper), "mobs", oper)
	case cmd == "fstat" && oper != "":
		txt = Fstat(oper)
	case cmd == "who" && oper != "":
		txt = append(txt, Who(oper))
	case cmd == "clist" && oper != "":
		txt = Clist(oper)
	case cmd == "char" && oper != "":
		txt = append(txt, CharInfo(oper))
	case cmd == "find" && oper != "":
		txt = append(txt, Find(oper))
	case cmd == "name" && oper != "":
		txt = append(txt, Name(oper))
	case cmd == "class" && oper != "":
		txt = FindClass(oper)
	case cmd == "delalt" && oper != "":
		txt = append(txt, DelAlt(oper, char))
	case cmd == "addalt" && oper != "":
		txt = append(txt, AddAlt(oper, char))
	case cmd == "addname" && oper != "":
		txt = append(txt, AddName(oper, char))
	case cmd == "lr":
		txt = LoadReport(oper, char)
	case cmd == "lrdel" && oper != "":
		txt = append(txt, LRDel(oper))
	default:
		txt = append(txt, BadSyntax)
		if guess := badCommand(cmd, oper); guess != "" {
			txt = append(txt, guess)
		}
	}
	var tells []string
	for _, t := range txt {
		for _, part := range splitTell(t) {
			// katumi.tin runs the line as a tintin command, and a reply can
			// repeat what a player sent: 404s, load reports, real names
			tells = append(tells, tellSafe("t "+char+" "+part)+"\n")
		}
	}
	return capTells(char, tells)
}

// maxReplyBytes is the most a reply to one tell sends the MUD. The MUD
// disconnects a character that sends it more than about 4 KB at once: 50
// tells of clist lines (about 3.8 KB) get through and 60 (4.6 KB) don't,
// while 80 short tells do. A clist or who of an account with dozens of alts
// runs past that, and so can class. The 1 KB to spare leaves room for
// anything else sent in the same moment. fstat's ten matches come to about
// 1 KB.
const maxReplyBytes = 3000

// capTells cuts a reply down to maxReplyBytes, counting each tell with the
// \r\n tintin ends it with. A reply cut short keeps as many tells as fit and
// ends with how many more there were.
func capTells(char string, tells []string) []string {
	used := 0
	for _, t := range tells {
		used += len(t) + 1
	}
	if used <= maxReplyBytes {
		return tells
	}
	var kept []string
	used = 0
	for i, t := range tells {
		// keep this tell only if the count of the ones after it still fits
		if used+len(t)+1+len(moreTells(char, len(tells)-i-1))+1 >
			maxReplyBytes {
			return append(kept, moreTells(char, len(tells)-i))
		}
		kept = append(kept, t)
		used += len(t) + 1
	}
	return kept
}

// moreTells is the tell that ends a reply cut short, saying how many more
// tells there were.
func moreTells(char string, more int) string {
	s := "s"
	if more == 1 {
		s = ""
	}
	return tellSafe(fmt.Sprintf(
		"t %s ...and %d more tell%s, too many to send at once.",
		char, more, s)) + "\n"
}

// tellLimit is the most text katumi puts in a single tell.
const tellLimit = 300

// splitTell breaks a reply into as many pieces as it takes to keep each one
// within tellLimit, breaking at a space where there is one.
func splitTell(t string) []string {
	var parts []string
	for len(t) > tellLimit {
		cut := strings.LastIndex(t[:tellLimit+1], " ")
		if cut <= 0 {
			cut = tellLimit // no space to break at
		}
		if part := strings.TrimRight(t[:cut], " "); part != "" {
			parts = append(parts, part)
		}
		t = strings.TrimLeft(t[cut:], " ")
	}
	if t != "" || len(parts) == 0 {
		parts = append(parts, t)
	}
	return parts
}
