package main

import (
	"database/sql"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test_Class checks class, which helps fill a group slot: the accounts
// online with a level 50 alt of a class, each with the char its player is
// online as and those alts, from the asker's side of the racewar and none
// of the asker's own.
func Test_Class(t *testing.T) {
	src, err := os.Getwd()
	chkErr(t, err)
	schema, err := os.ReadFile(filepath.Join(src, "init_db.sql"))
	chkErr(t, err)
	// a class that isn't one is logged as a 404
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

	// a clock that stands still, so each char was last seen exactly as long
	// ago as it says
	clock := time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)
	defer func(was func() time.Time) { now = was }(now)
	now = func() time.Time { return clock }

	for _, c := range []struct {
		acct, name, class, race string
		lvl                     int
		ago                     time.Duration // since last seen
		vis                     string
	}{
		// Gil asks from the good side, and has a cleric too
		{"Gil", "Gil", "Warrior", "Human", 50, 0, "t"},
		{"Gil", "Gilc", "Cleric", "Human", 50, 24 * time.Hour, "t"},
		// Ann is online as a warrior, with a cleric that counts, one a
		// level short, and a hidden one
		{"Ann", "Ann", "Warrior", "Human", 50, 10 * time.Second, "t"},
		{"Ann", "Annb", "Cleric", "Moon Elf", 50, 48 * time.Hour, "t"},
		{"Ann", "Annc", "Cleric", "Human", 49, 48 * time.Hour, "t"},
		{"Ann", "Annd", "Cleric", "Human", 50, 48 * time.Hour, "f"},
		// Bea is online as a cleric, and has a half-orc one, whose name
		// comes first
		{"Bea", "Bea", "Cleric", "Gnome", 50, 20 * time.Second, "t"},
		{"Bea", "Bab", "Cleric", "Half-Orc", 50, 72 * time.Hour, "t"},
		// Cal is online as an evil one
		{"Cal", "Cal", "Cleric", "Drow Elf", 50, 0, "t"},
		// Dee left two minutes ago
		{"Dee", "Dee", "Cleric", "Human", 50, 2 * time.Minute, "t"},
		// Eve is online, but only as a hidden alt
		{"Eve", "Eve", "Cleric", "Human", 50, 120 * time.Hour, "t"},
		{"Eve", "Evh", "Warrior", "Human", 20, 0, "f"},
		// Fay has just switched to Fax, so both were seen this minute
		{"Fay", "Fay", "Warrior", "Halfling", 40, 45 * time.Second, "t"},
		{"Fay", "Fax", "Warrior", "Halfling", 30, 5 * time.Second, "t"},
		{"Fay", "Faz", "Cleric", "Halfling", 50, 24 * time.Hour, "t"},
		// Ira asks from the evil side, and Hal as a half-orc
		{"Ira", "Ira", "Warrior", "Ogre", 50, 0, "t"},
		{"Hal", "Hal", "Warrior", "Half-Orc", 50, 0, "t"},
	} {
		_, err = db.Exec("INSERT OR IGNORE INTO accounts (account_name) "+
			"VALUES (?)", c.acct)
		chkErr(t, err)
		_, err = db.Exec("INSERT INTO chars VALUES (?, ?, ?, ?, ?, ?, ?)",
			c.acct, c.name, c.class, c.race, c.lvl, clock.Add(-c.ago), c.vis)
		chkErr(t, err)
	}

	// Accounts already on a cleric come first, then the rest by name, and
	// the cleric an account is on comes first in its list. Cal is evil, and
	// the rest are Gil's own, too low, hidden, or on accounts that aren't
	// online.
	good := []string{
		"Accounts online with a level 50 Cleric on the good side:",
		"@Bea, online as Bea: Bea (Gnome), Bab (Half-Orc)",
		"@Ann, online as Ann: Annb (Moon Elf)",
		"@Fay, online as Fax: Faz (Halfling)",
	}
	for _, name := range []string{"cleric", "Cleric", "CLE", "cler",
		"clerics", "cleric."} {
		chk(t, "class "+name+" from the good side", good,
			FindClass(name, "Gil"))
	}
	// Bea isn't on a cleric that's listed here
	chk(t, "class from the evil side", []string{
		"Accounts online with a level 50 Cleric on the evil side:",
		"@Cal, online as Cal: Cal (Drow Elf)",
		"@Bea, online as Bea: Bab (Half-Orc)",
	}, FindClass("cleric", "Ira"))
	// a half-orc, who could be on either side, sees both, and so does
	// someone Katumi can't see
	both := []string{
		"Accounts online with a level 50 Cleric:",
		"@Bea, online as Bea: Bea (Gnome), Bab (Half-Orc)",
		"@Cal, online as Cal: Cal (Drow Elf)",
		"@Ann, online as Ann: Annb (Moon Elf)",
		"@Fay, online as Fax: Faz (Halfling)",
		"@Gil, online as Gil: Gilc (Human)",
	}
	chk(t, "class from a half-orc", both, FindClass("cleric", "Hal"))
	chk(t, "class from someone", both, FindClass("cleric", "Someone"))

	chk(t, "class with none from the good side", []string{
		"No one else online has a level 50 Lich on the good side."},
		FindClass("lich", "Gil"))
	chk(t, "class with none from the evil side", []string{
		"No one else online has a level 50 Druid on the evil side."},
		FindClass("druid", "Ira"))
	chk(t, "class with none from a half-orc", []string{
		"No one else online has a level 50 Lich."}, FindClass("lich", "Hal"))
	chk(t, "class that isn't one", []string{
		"404 class not found: evil shaman"}, FindClass("evil shaman", "Gil"))

	// the other ways of naming a class
	for name, class := range map[string]string{
		"a-p": "Anti-Paladin", "AP": "Anti-Paladin",
		"anti paladin": "Anti-Paladin", "antipal": "Anti-Paladin",
		"dire raider": "Dire Raider", "dir": "Dire Raider",
		"dire": "Dire Raider", "necro": "Necromancer", "ench": "Enchanter",
		"bat": "Battlechanter", "bards": "Bard", "pal": "Paladin",
		"paladins": "Paladin", "sorc": "Sorcerer", "wizards": "Wizard",
		"illu": "Illusionist",
	} {
		if got, ok := classFor(db, name); !ok || got != class {
			t.Errorf("classFor() check failed: %q: Expected: %s, Actual: %q",
				name, class, got)
		}
	}
	// too short to tell Bard from Battlechanter, or no class at all
	for _, name := range []string{"", "ba", "en", "xyz", "evil shaman"} {
		if got, ok := classFor(db, name); ok {
			t.Errorf("classFor() check failed: %q: Expected: no class, "+
				"Actual: %s", name, got)
		}
	}
}
