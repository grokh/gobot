package main

import (
	"database/sql"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Pasted lists: the TorilEQ page looks up every item in a game command's
// output pasted into its box, and gobot -glist does the same for the lines
// torilmud.tin collects. Each kind of line the game prints has its own
// pattern here, taken from its real output. A line with no item in it, like
// a prompt, a header, a table's border, or the bids of an auction list
// wrapped onto lines of their own, is left out rather than looked up.

// shopLine is a line of a shop's list, the item before its price, which a
// mark can follow: " 3) a plate of chicken tarta for 1 silver, and 5 copper
// coins.", " 1) A double-bladed two-handed battleaxe for 5 platinum coins. (*)"
var shopLine = regexp.MustCompile(`^\d+\)\s+(.+?)\s+for\s+\d[\d,]*\s+` +
	`(?:platinum|gold|silver|copper)\b.*\bcoins?\.?(?:\s*\([^()]*\))?$`)

// auctionLine is a line of auction list: its number and the item's id, then
// the item, which the game lines up on the right, then the bid and the rest,
// unless the line was wrapped before them:
// "1)     100114               a pouch of rubbery deepspawn hide         100 ..."
var auctionLine = regexp.MustCompile(
	`^\d+\)\s+\d+\s+(.+?)(?:\s{2,}\d[\d,]*(?:\s.*)?)?$`)

// auctionRest is what a wrapped line of auction list leaves on lines of its
// own: "No Bid Yet  N/A  SELLER Name", "OPEN Name". No item's name has any of
// these in it.
var auctionRest = regexp.MustCompile(
	`No Bid Yet|\bN/A\b|\b(?:OPEN|CLOSED|ENDED|SELLER|BUYER)\b`)

// wornLine is a line of equipment, the item after the slot it's on:
// "<worn on head>       a star-metal chainmail hood (magic)". A prompt, like
// "< P: sit >", starts with a space instead.
var wornLine = regexp.MustCompile(`^<[a-z][a-z ]*>\s*(.*)$`)

// countPrefix is how inventory, a bag or storage shows several of an item:
// "[4] a feather"
var countPrefix = regexp.MustCompile(`^\[\s*\d+\]\s*`)

// listNames picks the item names out of pasted text, one per line that
// holds an item.
func listNames(txt string) []string {
	var names []string
	for _, line := range strings.Split(txt, "\n") {
		if name := lineName(line); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// lineName is the name of the item on a pasted line, or "" if it has none.
func lineName(line string) string {
	line = strings.TrimSpace(line)
	if m := lineItem(line); m != nil {
		line = m[1]
	} else if notItem(line) {
		return ""
	}
	// what the game shows after a name: (magic), (glowing) and the like
	if i := strings.Index(line, " ("); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	// glist puts who left an item in the cache before it, in a column
	if i := strings.LastIndex(line, "   "); i >= 0 {
		line = line[i:]
	}
	return strings.TrimSpace(countPrefix.ReplaceAllString(
		strings.TrimSpace(line), ""))
}

// lineItem matches a line of a shop's list, auction list or equipment, with
// the part naming the item as its first group, or is nil for any other line.
func lineItem(line string) []string {
	for _, re := range []*regexp.Regexp{shopLine, auctionLine, wornLine} {
		if m := re.FindStringSubmatch(line); m != nil {
			return m
		}
	}
	return nil
}

// notItem reports whether a line that isn't from a shop, auction list or
// equipment can't name an item: it has no letters (a table's border), or a
// colon (a header or prompt), a semicolon, a # (auction list's header), an
// h/ (a prompt) or [RETURN (the pager), none of which any item's name has,
// or it's the rest of a wrapped line of auction list.
func notItem(line string) bool {
	return !strings.ContainsFunc(line, unicode.IsLetter) ||
		strings.ContainsAny(line, ":;#") || strings.Contains(line, "h/") ||
		strings.Contains(line, "[RETURN") || line == "Auction list" ||
		auctionRest.MatchString(line)
}

// drinkIn is how a shop lists a drink: what's in it, then the container,
// which is the item: "White wine in a cup of Jhalessa's spiced wine".
var drinkIn = regexp.MustCompile(`^([A-Z][a-z]+)(?: [A-Za-z]+)*? in ` +
	`((?:a|an|the|some) .+)$`)

// pastedItem finds the item a pasted name means, as eqName does, and if
// that finds none, under the name as a shop shows it: with a capital first
// letter ("A thick black iron key"), or a drink in its container. It's ""
// if TorilDB.db has none of them.
func pastedItem(db *sql.DB, name string) (string, error) {
	tries := []string{name}
	if first, size := utf8.DecodeRuneInString(name); unicode.IsUpper(first) {
		tries = append(tries, string(unicode.ToLower(first))+name[size:])
	}
	if m := drinkIn.FindStringSubmatch(name); m != nil {
		switch m[1] {
		case "A", "An", "The", "Some": // an item's own name, not a drink
		default:
			tries = append(tries, m[2])
		}
	}
	for _, n := range tries {
		found, err := eqName(db, n)
		if found != "" || err != nil {
			return found, err
		}
	}
	return "", nil
}
