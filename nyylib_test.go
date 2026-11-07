package main

import (
	"database/sql"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// NyyLIB, the Mudlet package many TorilMUD players use
// (github.com/Nyyrazzilyss/NyyLIB), reads two kinds of line from Katumi with
// triggers: clist and char replies, which fill its who list, and the NHC
// welcome, which makes it look the newcomer up. Their wording is a promise
// to NyyLIB's players, not only to Test_All. These are its patterns, copied
// from its triggers in NyyLIB013dev.mpackage (2025-11-13).
var (
	// "Katumi tells you": whoadd(name, class, race, account), from groups
	// 2, 1, 3 and 4
	nyyCharLine = regexp.MustCompile(`Katumi tells you '\[[0-9]+ (.*)\] ` +
		`([A-Za-z]+) \((.*)\) \(@([A-Za-z]+)\) seen .*'`)
	// "Katumi NHC: (H) 'Welcome,": who on group 1
	nyyWelcome = regexp.MustCompile(`^(?:< .* > )?(?:@[A-Z][a-z]+/)?` +
		`Katumi NHC: .* 'Welcome, ([A-Za-z]+).*`)
)

// Test_NyyLIB checks that NyyLIB's triggers still read Katumi's clist and
// char replies and its NHC welcome, and read them right.
func Test_NyyLIB(t *testing.T) {
	src, err := os.Getwd()
	chkErr(t, err)
	schema, err := os.ReadFile(filepath.Join(src, "init_db.sql"))
	chkErr(t, err)
	// who() logs each new account and char
	defer log.SetOutput(log.Writer())
	log.SetOutput(io.Discard)

	// a fresh toril.db in a temp folder, made with the driver, so this test
	// needs no sqlite3 CLI
	chkErr(t, os.Chdir(t.TempDir()))
	defer os.Chdir(src)
	db, err := sql.Open("sqlite3", "toril.db")
	chkErr(t, err)
	_, err = db.Exec(string(schema))
	chkErr(t, err)
	chkErr(t, db.Close())

	// A newcomer on a new account, who gets the welcome, then alts with
	// two-word and hyphenated classes and races, added the way who lines
	// add them.
	var welcomes []string
	for _, c := range []Char{
		{name: "Tom", lvl: 1, class: "Warrior ", race: "Drow Elf", acct: "Bob"},
		{name: "Bob", lvl: 50, class: "Dire Raider ", race: "Half-Elf",
			acct: "Bob"},
		{name: "Zed", lvl: 37, class: "Anti-Paladin ", race: "Shield Dwarf",
			acct: "Bob"},
	} {
		welcomes = append(welcomes, c.who()...)
	}

	// The MUD shows Katumi's NHC lines to others with an (H) tag, as
	// NyyLIB's own sample of this line has it. Its pattern needs a tag there.
	if len(welcomes) != 1 || !strings.HasPrefix(welcomes[0], "nhc ") {
		t.Fatalf("who() check failed: Expected: one nhc welcome, Actual: %q",
			welcomes)
	}
	line := "Katumi NHC: (H) '" + strings.TrimPrefix(welcomes[0], "nhc ") + "'"
	if m := nyyWelcome.FindStringSubmatch(line); m == nil || m[1] != "Tom" {
		t.Errorf("NyyLIB welcome check failed: %q should make it run who Tom, "+
			"Actual: %q", line, m)
	}

	// clist lists the account's chars by level, then name
	chkCharLines(t, "clist tom", heard(t, "Yog", ReplyTo("Yog", "clist tom")),
		[][4]string{
			{"Dire Raider", "Bob", "Half-Elf", "Bob"},
			{"Anti-Paladin", "Zed", "Shield Dwarf", "Bob"},
			{"Warrior", "Tom", "Drow Elf", "Bob"},
		})
	chkCharLines(t, "char zed", heard(t, "Yog", ReplyTo("Yog", "char zed")),
		[][4]string{{"Anti-Paladin", "Zed", "Shield Dwarf", "Bob"}})
}

// heard is a reply from ReplyTo as the asker sees it: the MUD shows the
// tell "t asker text" to them as "Katumi tells you 'text'".
func heard(t *testing.T, asker string, tells []string) []string {
	var lines []string
	for _, tell := range tells {
		text, ok := strings.CutPrefix(strings.TrimSuffix(tell, "\n"),
			"t "+asker+" ")
		if !ok {
			t.Errorf("ReplyTo() check failed: %q isn't a tell to %s", tell, asker)
			continue
		}
		lines = append(lines, "Katumi tells you '"+text+"'")
	}
	return lines
}

// chkCharLines checks that NyyLIB's trigger reads each line as the class,
// name, race and account in want, in order.
func chkCharLines(t *testing.T, what string, lines []string,
	want [][4]string) {
	if len(lines) != len(want) {
		t.Errorf("NyyLIB %s check failed: Expected: %d lines, Actual: %q",
			what, len(want), lines)
		return
	}
	for i, line := range lines {
		m := nyyCharLine.FindStringSubmatch(line)
		if m == nil {
			t.Errorf("NyyLIB %s check failed: its trigger doesn't match %q",
				what, line)
			continue
		}
		if got := [4]string{m[1], m[2], m[3], m[4]}; got != want[i] {
			t.Errorf("NyyLIB %s check failed: %q reads as %q, not %q",
				what, line, got, want[i])
		}
	}
}
