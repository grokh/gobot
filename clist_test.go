package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test_Clist checks clist's filters: a class, a level, or both after the
// name list only the account's alts that fit, in the same lines as a whole
// clist. With the same accounts, it checks which account name answers for
// when a name means two.
func Test_Clist(t *testing.T) {
	src, err := os.Getwd()
	chkErr(t, err)
	schema, err := os.ReadFile(filepath.Join(src, "init_db.sql"))
	chkErr(t, err)
	// a name or class that isn't one is logged as a 404
	defer log.SetOutput(log.Writer())
	log.SetOutput(io.Discard)

	// a fresh toril.db in a temp folder, made with the driver, so this test
	// needs no sqlite3 CLI
	chkErr(t, os.Chdir(t.TempDir()))
	defer os.Chdir(src)
	db, err := sql.Open("sqlite3", "toril.db")
	chkErr(t, err)
	// closed before the temp folder is removed: Windows won't delete an
	// open file
	defer db.Close()
	_, err = db.Exec(string(schema))
	chkErr(t, err)

	const seen = "2026-11-14 20:00:00"
	type alt struct {
		acct, name, class, race string
		lvl                     int
		vis                     string
	}
	alts := []alt{
		{"Bob", "Bob", "Cleric", "Human", 50, "t"},
		{"Bob", "Bab", "Warrior", "Human", 50, "t"},
		{"Bob", "Bec", "Cleric", "Gnome", 45, "t"},
		{"Bob", "Bid", "Dire Raider", "Orc", 40, "t"},
		{"Bob", "Bod", "Cleric", "Human", 30, "t"},
		{"Bob", "Byz", "Anti-Paladin", "Drow Elf", 1, "t"},
		// hidden, so never listed
		{"Bob", "Bud", "Cleric", "Human", 50, "f"},
		// an account named like a class
		{"Ranger", "Ranger", "Ranger", "Half-Elf", 50, "t"},
		{"Ranger", "Rex", "Warrior", "Human", 50, "t"},
		// an account named like another account's char, so clist cal lists
		// both
		{"Cal", "Cid", "Warrior", "Human", 50, "t"},
		{"Dot", "Cal", "Cleric", "Human", 40, "t"},
		// the same again, where the account named like the char is the newer
		// of the two but the first by name, for name
		{"Zoe", "Amy", "Warrior", "Human", 20, "t"},
		{"Amy", "Ava", "Cleric", "Human", 10, "t"},
	}
	line := map[string]string{} // each alt's clist line, by name
	for _, a := range alts {
		_, err = db.Exec("INSERT OR IGNORE INTO accounts (account_name) "+
			"VALUES (?)", a.acct)
		chkErr(t, err)
		_, err = db.Exec("INSERT INTO chars VALUES (?, ?, ?, ?, ?, ?, ?)",
			a.acct, a.name, a.class, a.race, a.lvl, seen, a.vis)
		chkErr(t, err)
		line[a.name] = fmt.Sprintf("[%d %s] %s (%s) (@%s) seen %s UTC",
			a.lvl, a.class, a.name, a.race, a.acct, seen)
	}
	lines := func(names ...string) []string {
		var txt []string
		for _, name := range names {
			txt = append(txt, line[name])
		}
		return txt
	}

	for opers, good := range map[string][]string{
		// all of them, highest level first, then by name
		"bob|BAB|bob 0+|bob level": lines(
			"Bab", "Bob", "Bec", "Bid", "Bod", "Byz"),
		// a class, named any way class takes one
		"bob cleric|bab Cleric|bob cle|bob CLER|bob clerics": lines(
			"Bob", "Bec", "Bod"),
		"bob dire raider|bob dire|bob dir": lines("Bid"),
		"bob a-p|bob anti paladin|bob AP":  lines("Byz"),
		// a level, after level or lvl or not
		"bob 50|bob level 50|bob lvl 50": lines("Bab", "Bob"),
		// a range, given either way round, and a lowest
		"bob 40-50|bob 50-40|bob 40 50|bob level 50 40": lines(
			"Bab", "Bob", "Bec", "Bid"),
		"bob 45+|bob lvl 45+": lines("Bab", "Bob", "Bec"),
		// both, in either order
		"bob cleric 50|bob 50 cleric|bob cle lvl 50": lines("Bob"),
		"bob cleric 31+|bob 31-50 cle":               lines("Bob", "Bec"),
		"bob 1 a-p|bob a-p 1-10":                     lines("Byz"),
		// the name comes first, so an account can be named like a class
		"ranger|rex": lines("Ranger", "Rex"),
		"ranger ranger|rex ran 50|ranger rangers": lines("Ranger"),
		// a name that is one account's and another's char
		"cal|cal 40+":   lines("Cid", "Cal"),
		"cal cleric":    lines("Cal"),
		"cal 50|cid 1+": lines("Cid"),
		"dot":           lines("Cal"),

		// an account that has none of them
		"bob cleric 40|bab Cleric 40": {"@Bob has no level 40 Cleric."},
		"bob druid":                   {"@Bob has no Druid."},
		"bob 2-29|bob 29 2":           {"@Bob has no alt of level 2 to 29."},
		"bob 51+":                     {"@Bob has no alt of level 51 or higher."},
		"bob druid 45+":               {"@Bob has no Druid of level 45 or higher."},
		"bob 0":                       {"@Bob has no level 0 alt."},
		"ranger cleric":               {"@Ranger has no Cleric."},
		"cal druid":                   {"@Cal and @Dot have no Druid."},
		"dot druid":                   {"@Dot has no Druid."},

		// a class that isn't one, or a name that isn't anyone's
		"bob xyz|bob xyz 50|bob 50 xyz": {"404 class not found: xyz"},
		"bob evil shaman 50":            {"404 class not found: evil shaman"},
		"bob 1000":                      {"404 class not found: 1000"},
		"bob ba":                        {"404 class not found: ba"},
		"nobody|nobody cleric|nobody xyz": {
			"404 character or account not found: nobody"},
		"bud cleric": {"404 character or account not found: bud"},
	} {
		for _, oper := range strings.Split(opers, "|") {
			chk(t, "Clist("+oper+")", good, Clist(oper))
		}
	}

	// name looks the account up the way clist does. For a name that is one
	// account's and another's char, it answers for the one with a real name
	// on record. Of two with none it's the first by name: Amy, though the DB
	// has had Zoe longer. Of two with one it's the one the DB has had
	// longer: Zoe. Each case first gives acct the real name player, if any.
	for _, c := range []struct{ acct, player, oper, good string }{
		{"", "", "cal", "@Cal did not disclose their real name"},
		{"Dot", "Dee", "cal", "@Dot's real name is Dee"},
		{"Cal", "Cy", "cal", "@Cal's real name is Cy"},
		{"", "", "dot", "@Dot's real name is Dee"},
		{"", "", "CID", "@Cal's real name is Cy"},
		{"", "", "amy", "@Amy did not disclose their real name"},
		{"", "", "ava", "@Amy did not disclose their real name"},
		{"Zoe", "Zia", "amy", "@Zoe's real name is Zia"},
		{"Amy", "Ada", "amy", "@Zoe's real name is Zia"},
		{"", "", "ava", "@Amy's real name is Ada"},
		{"", "", "bob", "@Bob did not disclose their real name"},
		// a hidden char's name finds no account
		{"", "", "bud", "404 character or account not found: bud"},
	} {
		if c.acct != "" {
			_, err = db.Exec("UPDATE accounts SET player_name = ? "+
				"WHERE account_name = ?", c.player, c.acct)
			chkErr(t, err)
		}
		chk(t, "Name("+c.oper+")", []string{c.good}, []string{Name(c.oper)})
	}

	// through a tell, as Katumi sends it
	chk(t, "clist with a class and a level",
		[]string{"t Yog " + line["Bob"] + "\n"},
		ReplyTo("Yog", "clist bob cleric 50"))
	chk(t, "clist with none of a class",
		[]string{"t Yog @Bob has no Druid.\n"},
		ReplyTo("Yog", "clist bob druid"))

	// help clist tells how, each entry in a tell of its own
	tells := ReplyTo("Yog", "help clist")
	help := strings.Join(tells, "")
	for _, want := range []string{"clist <acct/char> [class] [level]",
		"clist rynshana cleric 50", "(40-50)", "(45+)"} {
		if len(tells) != 2 || !strings.Contains(help, want) {
			t.Errorf("help clist check failed: no %q in 2 tells: %q",
				want, tells)
		}
	}
}
