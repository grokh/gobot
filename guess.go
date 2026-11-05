package main

import (
	"database/sql"
	"log"
	"sort"
	"strings"
)

// Katumi's guesses at what a tell meant (issue #22): the command a mistyped
// one is a typo of, and the names a stat, astat or mstat that found nothing
// was likely after. They're only ever suggestions: a guessed command is
// never run, since some of them change what Katumi keeps.

// commands are the tell commands guessCommand picks from.
var commands = []string{"stat", "astat", "fstat", "mstat", "who", "clist",
	"char", "find", "class", "name", "addname", "delalt", "addalt", "hidden",
	"help", "lr", "lrdel"}

// typoEdits is how many edits a word can be from the one it's taken for a
// typo of: one for a word of up to six letters, two for a longer one. So
// stats is stat, but hello isn't help.
func typoEdits(word string) int {
	if len([]rune(word)) < 7 {
		return 1
	}
	return 2
}

// edits is how many letters it takes to add, drop, change, or swap with the
// next one, to make a into b: the optimal string alignment distance, so a
// swap like stdu for stud is one edit, not two.
func edits(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			change := 1
			if ra[i-1] == rb[j-1] {
				change = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+change)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(ra)][len(rb)]
}

// guessCommand is the command cmd is closest to, if it's near enough to be
// a typo of it: stats is stat and fsat fstat. It's "" if cmd is near none.
func guessCommand(cmd string) string {
	cmd = strings.ToLower(cmd)
	best, most := "", typoEdits(cmd)
	for _, c := range commands {
		if e := edits(cmd, c); e <= most {
			best, most = c, e-1
		}
	}
	return best
}

// badCommand is the reply's line after BadSyntax for a tell whose command
// Katumi doesn't take: the tell with the command it looks like a typo of,
// or the command's help, if all it lacks is what to look up. It's "" if
// there's nothing to suggest.
func badCommand(cmd string, oper string) string {
	guess := guessCommand(cmd)
	switch {
	case guess == "":
		return ""
	case guess == strings.ToLower(cmd):
		return "See 'tell katumi help " + guess + "'."
	}
	return "Did you mean 'tell katumi " + strings.TrimSpace(guess+" "+oper) +
		"'?"
}

// withGuess adds a line to a name lookup's reply that's only its 404: the
// names in table (eq or mobs) that oper was likely after, if there are any.
func withGuess(txt []string, table string, oper string) []string {
	if len(txt) != 1 || !strings.HasPrefix(txt[0], "404 ") {
		return txt
	}
	db, err := OpenItemDB()
	if err != nil {
		return txt
	}
	defer db.Close()
	guess, err := guessNames(db, table, oper)
	if err != nil {
		log.Printf("%s: guessing at %s: %v\n", itemDBFile, oper, err)
		return txt
	}
	if guess != "" {
		txt = append(txt, tintinSafe(guess))
	}
	return txt
}

// guessLimit is the most names a guess offers.
const guessLimit = 3

// guessNames is the line suggesting what a lookup of oper in table (eq or
// mobs) that found nothing was after. Each of oper's words that's in no
// name is taken for a typo of the word in the names closest to it, the
// commonest if several are as close, and the line offers the names that
// have every word once those are fixed, the shortest first, as stat would.
// Mobs go by their names without Kegor's numbers. It's "" if a word is
// near no word in the names, if none needed fixing, or if the fixed words
// still match nothing.
func guessNames(db *sql.DB, table string, oper string) (string, error) {
	rows, err := db.Query("SELECT name FROM " + table)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var names, lower []string
	seen := make(map[string]bool)
	used := make(map[string]int) // how many names each word is in
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", err
		}
		if table == "mobs" { // the bartender 12 is the bartender
			name = strings.TrimRight(strings.TrimRight(name, "0123456789"), " ")
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
		lower = append(lower, strings.ToLower(name))
		for _, w := range strings.Fields(lower[len(lower)-1]) {
			if w = strings.Trim(w, `,.;:!?()"`); w != "" {
				used[w]++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	words := strings.Fields(strings.ToLower(oper))
	fixed := false
	for n, w := range words {
		if inAny(lower, w) {
			continue
		}
		most := typoEdits(w)
		best, bestEdits := "", most+1
		for v, count := range used {
			if len(v) < len(w)-most || len(v) > len(w)+most {
				continue
			}
			e := edits(w, v)
			closer := e < bestEdits
			asClose := e == bestEdits && best != "" &&
				(count > used[best] || count == used[best] && v < best)
			if closer || asClose {
				best, bestEdits = v, e
			}
		}
		if best == "" {
			return "", nil
		}
		words[n] = best
		fixed = true
	}
	if !fixed {
		return "", nil
	}

	var guesses []string
	for n, l := range lower {
		if hasAll(l, words) {
			guesses = append(guesses, names[n])
		}
	}
	if len(guesses) == 0 {
		return "", nil
	}
	sort.Slice(guesses, func(a, b int) bool {
		if len(guesses[a]) != len(guesses[b]) {
			return len(guesses[a]) < len(guesses[b])
		}
		return guesses[a] < guesses[b]
	})
	if len(guesses) > guessLimit {
		guesses = guesses[:guessLimit]
	}
	return "Did you mean " + orList(guesses) + "?", nil
}

// inAny is whether any of names has word in it.
func inAny(names []string, word string) bool {
	for _, n := range names {
		if strings.Contains(n, word) {
			return true
		}
	}
	return false
}

// hasAll is whether name has every one of words in it.
func hasAll(name string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(name, w) {
			return false
		}
	}
	return true
}

// orList is a list in words: a, a or b, a, b or c.
func orList(items []string) string {
	if len(items) == 1 {
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " or " +
		items[len(items)-1]
}
