package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func chk(t *testing.T, check string, good []string, txt []string) {
	if len(txt) == len(good) {
		for i := range txt {
			if txt[i] != good[i] {
				t.Errorf(
					"%s check failed: Expected: %s, Actual: %s",
					check, strconv.Quote(good[i]), strconv.Quote(txt[i]))
			}
		}
	} else {
		t.Errorf(
			"%s check failed: Expected: %d responses, Actual: %d",
			check, len(good), len(txt))
	}
}

func chkErr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Check errored: %s", err)
	}
}

func Test_All(t *testing.T) {
	src, err := os.Getwd()
	chkErr(t, err)
	chkErr(t, os.MkdirAll(filepath.Join(src, "logs"), 0750))
	f, err := os.OpenFile(
		filepath.Join(src, "logs", "test.log"),
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0640,
	)
	chkErr(t, err)
	defer f.Close()
	log.SetOutput(f)

	// Run in a temp folder with a fresh toril.db, so the test never touches
	// the toril.db, TorilDB.db or import.sql in the folder it's run from.
	chkErr(t, os.Chdir(t.TempDir()))
	defer os.Chdir(src)
	cmd := exec.Command("sqlite3", "toril.db",
		".read "+strconv.Quote(filepath.Join(src, "init_db.sql")))
	chkErr(t, cmd.Run())

	// A clock that starts on a whole second and goes a millisecond further
	// each time it's read, so the times gobot stamps on what it records all
	// fall in the second date names. Read from a running clock, they'd be a
	// second on whenever one ticked over mid-test. They still differ, as two
	// load reports at the same time are one too many, and by enough for
	// find, which sorts by JULIANDAY, a day count too coarse for microseconds.
	start := time.Now().Truncate(time.Second)
	reads := 0
	now = func() time.Time {
		reads++
		return start.Add(time.Duration(reads) * time.Millisecond)
	}
	defer func() {
		now = time.Now
		if reads >= 1000 {
			t.Errorf("Test_All's clock passed date's second: %d reads", reads)
		}
	}()
	date := start.UTC().Format("2006-01-02 15:04:05")

	// test output from 'time' command - run at connect and every 30s, here
	// timed by gobot's own clock, as when -seen is left out
	up := "0:01:30"
	Uptime(up, seenAt(0))

	// closed before the temp folder is removed: Windows won't delete an
	// open file
	db := OpenDB()
	defer db.Close()
	query := "SELECT uptime FROM boots WHERE boot_id = 1"
	stmt, err := db.Prepare(query)
	chkErr(t, err)
	defer stmt.Close()

	txt := make([]string, 1)
	err = stmt.QueryRow().Scan(&txt[0])
	chkErr(t, err)
	good := []string{"0:01:30"}
	chk(t, "Uptime()", good, txt)

	// test output from 'who' command - run every 30s
	who := "[50 Sha] Yog  (Barbarian)|" +
		"[50 Bar] Bob - Soul Singer - The Warders (Human)|" +
		"[ 1 War] Tom  (Drow Elf)"
	good = []string{"who Yog\n", "who Bob\n", "who Tom\n"}
	txt = WhoBatch(who)
	chk(t, "WhoBatch()", good, txt)

	// test output from 'who char' command - run whenever new char spotted
	c := Char{
		name: "Yog", lvl: 50, class: "Shaman ", race: "Barbarian", acct: "Yog",
	}
	txt = c.who()
	good = []string{}
	chk(t, "c.who()", good, txt)

	c = Char{
		name: "Tom", lvl: 1, class: "Warrior ", race: "Drow Elf", acct: "Bob",
	}
	txt = c.who()
	good = []string{
		"nhc Welcome, Tom. If you have any questions, " +
			"feel free to ask on this channel.",
	}
	chk(t, "c.who()", good, txt)

	c = Char{
		name: "Bob", lvl: 50, class: "Bard      ", race: "Human", acct: "Bob",
	}
	txt = c.who()
	good = []string{}
	chk(t, "c.who()", good, txt)

	query = "SELECT count(*) FROM chars"
	stmt, err = db.Prepare(query)
	chkErr(t, err)
	defer stmt.Close()

	txt = make([]string, 1)
	err = stmt.QueryRow().Scan(&txt[0])
	chkErr(t, err)
	good = []string{"3"}
	chk(t, "Who()", good, txt)

	// test output from tells
	char, tell := "Yog", "blah"
	txt = ReplyTo(char, tell)
	good = []string{
		"t Yog Invalid syntax. For valid syntax: tell katumi ?, " +
			"tell katumi help <cmd>\n",
	}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "?"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog I am a Helper Bot (Beta). Each command I accept " +
		"has further help files available at: tell katumi help <cmd>\n",
		"t Yog Find items: Acronymed stats: stat <item name>, " +
			"Stats fully spelled out: astat <item name>, " +
			"Find items by attributes, slots, etc.: (for proper usage, " +
			"tell katumi help fstat)\n",
		"t Yog Find mobs: Stats and the eq they carry: mstat <mob name>\n",
		"t Yog Find people: Acct and char info: who <char/acct>, " +
			"clist <char/acct> [class] [level], char <char>, " +
			"Show last online alt: " +
			"find <char/acct>, Find level 50 alts of a class among people " +
			"online: class <class>, RL names: name <char/acct>, " +
			"addname <name>, Control your listing: delalt <char>, " +
			"addalt <char>\n",
		"t Yog Misc: This message: ?, More info on each command: help <cmd>, " +
			"Find out if you're hidden: hidden, Load reports for rares or " +
			"global mobs: lr, lr <report>, lrdel <num>\n",
	}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "help"
	txt = ReplyTo(char, tell)
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "hidden"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog Yog is NOT hidden!\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Someone", "hidden"
	txt = ReplyTo(char, tell)
	good = []string{}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "who blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 character or account not found: blah\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "who bob"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog @Bob: Bob, Tom\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "char Blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 character not found: Blah\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "char bob"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog [50 Bard] Bob (Human) (@Bob) seen " + date + " UTC\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "clist bob"
	txt = ReplyTo(char, tell)
	good = []string{
		"t Yog [50 Bard] Bob (Human) (@Bob) seen " + date + " UTC\n",
		"t Yog [1 Warrior] Tom (Drow Elf) (@Bob) seen " + date + " UTC\n",
	}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "clist blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 character or account not found: blah\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "class blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 class not found: blah\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	// Yog is a Barbarian, so on the good side
	char, tell = "Yog", "class enchanter"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog No one else online has a level 50 Enchanter " +
		"on the good side.\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "class bard"
	txt = ReplyTo(char, tell)
	good = []string{
		"t Yog Accounts online with a level 50 Bard on the good side:\n",
		"t Yog @Bob, online as Bob: Bob (Human)\n",
	}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "lr"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog No loads reported for current boot.\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "lrdel 1"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog No loads reported for current boot.\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "lr blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog Invalid syntax. For valid syntax:" +
		" tell katumi ?, tell katumi help <cmd>\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "lr ogre in space"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog Load reported: ogre in space\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "lr thing at place"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog Load reported: thing at place\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "lr"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 1: ogre in space [Yog at " + date + " UTC]\n",
		"t Yog 2: thing at place [Yog at " + date + " UTC]\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "lrdel 2"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog Load deleted: thing at place\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "lrdel 3"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog Invalid load report number.\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "lrdel blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog Invalid syntax. For valid syntax: " +
		"tell katumi ?, tell katumi help <cmd>\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "find bob"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog @Bob is online as Bob\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "find blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 character or account not found: blah\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "delalt tom"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 character or account not found: tom\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Bob", "delalt tom"
	txt = ReplyTo(char, tell)
	good = []string{"t Bob Removed character from your alt list:: tom\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Bob", "who tom"
	txt = ReplyTo(char, tell)
	good = []string{"t Bob 404 character or account not found: tom\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "addalt tom"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 character or account not found: tom\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Bob", "addalt tom"
	txt = ReplyTo(char, tell)
	good = []string{"t Bob Re-added character to your alt list:: tom\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "name bob"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog @Bob did not disclose their real name\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Bob", "addname Bob"
	txt = ReplyTo(char, tell)
	good = []string{"t Bob Your real name recorded as: Bob\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "name bob"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog @Bob's real name is Bob\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "name blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 character or account not found: blah\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "help ?"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog Syntax: tell katumi ? -- " +
		"Katumi provides a full listing of valid commands.\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "help blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 help file not found: blah\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	testHelp(t)
	testCharChanges(t)
	testWhoWaits(t, db)
	testNewClasses(t, db)

	testItems(t, src)
	testReboot(t)
	testOnline(t)
	testSplit(t)
	testTellCap(t, db)
	testTimes(t, db, filepath.Join(src, "fix_times_and_stats.sql"))
	testRestore(t, db, testBackup(t))
	testRecordsDown(t, db) // last: it breaks toril.db's writes for a while
}

// chkPrefix checks that a one-line reply starts with want, for replies that
// end in a time that can tick over while the test runs.
func chkPrefix(t *testing.T, check string, want string, txt []string) {
	if len(txt) != 1 || !strings.HasPrefix(txt[0], want) {
		t.Errorf("%s check failed: Expected: %s..., Actual: %q",
			check, strconv.Quote(want), txt)
	}
}

// testCharChanges checks that class, race and level changes in the game
// reach the DB, and which of them ask for a lookup, starting from Yog as a
// Barbarian Shaman.
func testCharChanges(t *testing.T) {
	who := "[50 War] Yog  (Barbarian)|" +
		"[50 Bar] Bob - Soul Singer - The Warders (Human)|" +
		"[ 1 Xyz] Tom  (Drow Elf)"
	// Yog's class changed and Bob's didn't; Xyz isn't a class abbreviation
	// the DB knows, so Tom is left alone
	chk(t, "WhoBatch() class change", []string{"who Yog\n"}, WhoBatch(who))
	chkPrefix(t, "char after class change",
		"t Yog [50 Warrior] Yog (Barbarian) (@Yog) seen ",
		ReplyTo("Yog", "char yog"))
	// the DB now matches the who list, so Yog isn't flagged again
	chk(t, "WhoBatch() same classes", []string{}, WhoBatch(who))

	// the individual who that follows picks up a new race
	c := Char{
		name: "Yog", lvl: 50, class: "Warrior   ", race: "Half-Elf", acct: "Yog",
	}
	chk(t, "c.who() race change", []string{}, c.who())
	chkPrefix(t, "char after race change",
		"t Yog [50 Warrior] Yog (Half-Elf) (@Yog) seen ",
		ReplyTo("Yog", "char yog"))

	// a race the races table doesn't have is skipped, not written
	c = Char{
		name: "Yog", lvl: 50, class: "Warrior", race: "Martian", acct: "Yog",
	}
	chk(t, "c.who() unknown race", []string{}, c.who())
	chkPrefix(t, "char after unknown race",
		"t Yog [50 Warrior] Yog (Half-Elf) (@Yog) seen ",
		ReplyTo("Yog", "char yog"))

	// the who list showing another race, as when a char is deleted and made
	// again under its name: it's recorded from the list, and an individual
	// who asked for, which would catch a new account
	chk(t, "WhoBatch() race change", []string{"who Yog\n"},
		WhoBatch("[50 War] Yog  (Human)"))
	chkPrefix(t, "char after WhoBatch() race change",
		"t Yog [50 Warrior] Yog (Human) (@Yog) seen ",
		ReplyTo("Yog", "char yog"))
	chk(t, "WhoBatch() same race", []string{},
		WhoBatch("[50 War] Yog  (Human)"))
	// the race in another case is the same race, and what isn't a race,
	// like a flag after it, is left alone rather than looked up every time
	chk(t, "WhoBatch() race in lower case", []string{},
		WhoBatch("[50 War] Yog  (human)"))
	chk(t, "WhoBatch() flag after the race", []string{},
		WhoBatch("[50 War] Yog  (Human) (Linkdead)"))
	chkPrefix(t, "char after WhoBatch() flag after the race",
		"t Yog [50 Warrior] Yog (Human) (@Yog) seen ",
		ReplyTo("Yog", "char yog"))

	// a level that fell is a char started over: looked up once, and the
	// levels it gains after that are nothing new
	chk(t, "WhoBatch() level drop", []string{"who Yog\n"},
		WhoBatch("[ 1 War] Yog  (Human)"))
	chk(t, "WhoBatch() same level", []string{},
		WhoBatch("[ 1 War] Yog  (Human)"))
	chk(t, "WhoBatch() level up", []string{},
		WhoBatch("[50 War] Yog  (Human)"))

	// back to a Half-Elf for the checks that follow
	chk(t, "WhoBatch() race change back", []string{"who Yog\n"},
		WhoBatch("[50 War] Yog  (Half-Elf)"))

	testAcctChanges(t)
}

// testAcctChanges checks that a char whose who line shows another account
// than the DB has moves to it, starting and ending with Tom, a level 1 Drow
// Elf Warrior, on Bob's account.
func testAcctChanges(t *testing.T) {
	// on Yog's account, which the DB has and spells with a capital: Tom
	// moves, nobody is welcomed and Bob's alts no longer include him
	c := Char{
		name: "Tom", lvl: 1, class: "Warrior   ", race: "Drow Elf", acct: "yog",
	}
	chk(t, "c.who() account change", []string{}, c.who())
	chkPrefix(t, "char after account change",
		"t Yog [1 Warrior] Tom (Drow Elf) (@Yog) seen ",
		ReplyTo("Yog", "char tom"))
	chkPrefix(t, "old account's clist after account change",
		"t Yog [50 Bard] Bob (Human) (@Bob) seen ",
		ReplyTo("Yog", "clist bob"))
	if txt := ReplyTo("Bob", "clist yog"); len(txt) != 2 {
		t.Errorf("new account's clist check failed: %q", txt)
	}
	// seen again on the same account, he stays
	chk(t, "c.who() same account", []string{}, c.who())
	chkPrefix(t, "char after same account",
		"t Yog [1 Warrior] Tom (Drow Elf) (@Yog) seen ",
		ReplyTo("Yog", "char tom"))

	// a who line read wrong, its account running on into what followed it:
	// no account is made of that, and he stays where he is
	c = Char{
		name: "Tom", lvl: 1, class: "Warrior   ", race: "Drow Elf",
		acct: "Bob) (Half-Orc",
	}
	chk(t, "c.who() misread account", []string{}, c.who())
	chkPrefix(t, "char after misread account",
		"t Yog [1 Warrior] Tom (Drow Elf) (@Yog) seen ",
		ReplyTo("Yog", "char tom"))
	chk(t, "misread account not added",
		[]string{"t Yog 404 character or account not found: Bob) (Half-Orc\n"},
		ReplyTo("Yog", "who Bob) (Half-Orc"))
	// a light flag after the account is the MUD's, and read past; an account
	// written into a title isn't, so the line's last one is the one that
	// counts. Either way he's still Yog's.
	for _, acct := range []string{"yog)(In Dark", "Yog) (Daylight",
		"Bob) (Drow Elf) (@Yog", "Bob) (Drow Elf) (@yog)(In Dark"} {
		c.acct = acct
		chk(t, "c.who() account "+acct, []string{}, c.who())
		chkPrefix(t, "char after account "+acct,
			"t Yog [1 Warrior] Tom (Drow Elf) (@Yog) seen ",
			ReplyTo("Yog", "char tom"))
	}

	// Yog hides him, and he turns up as a Human on an account new to the DB:
	// a new player's low-level char, so they're welcomed, and the new race is
	// recorded with the new account
	chk(t, "delalt before account change",
		[]string{"t Yog Removed character from your alt list:: tom\n"},
		ReplyTo("Yog", "delalt tom"))
	c = Char{
		name: "Tom", lvl: 1, class: "Warrior   ", race: "Human", acct: "Zed",
	}
	chk(t, "c.who() new account", []string{
		"nhc Welcome, Tom. If you have any questions, " +
			"feel free to ask on this channel.",
	}, c.who())
	// he's still hidden, and only his new account can change that
	chk(t, "char hidden after account change",
		[]string{"t Yog 404 character not found: tom\n"},
		ReplyTo("Yog", "char tom"))
	chk(t, "old account's addalt after account change",
		[]string{"t Yog 404 character or account not found: tom\n"},
		ReplyTo("Yog", "addalt tom"))
	chk(t, "new account's addalt after account change",
		[]string{"t Tom Re-added character to your alt list:: tom\n"},
		ReplyTo("Tom", "addalt tom"))
	chkPrefix(t, "char after new account",
		"t Yog [1 Warrior] Tom (Human) (@Zed) seen ",
		ReplyTo("Yog", "char tom"))

	// back on Bob's account as a Drow Elf, where the checks after this one
	// expect him
	c = Char{
		name: "Tom", lvl: 1, class: "Warrior   ", race: "Drow Elf", acct: "Bob",
	}
	chk(t, "c.who() account change back", []string{}, c.who())
	chk(t, "clist after account change back", []string{
		"t Yog [50 Bard] Bob (Human) (@Bob) seen ",
		"t Yog [1 Warrior] Tom (Drow Elf) (@Bob) seen ",
	}, func() []string {
		txt := ReplyTo("Yog", "clist bob")
		for i := range txt {
			if j := strings.Index(txt[i], "seen "); j >= 0 {
				txt[i] = txt[i][:j+len("seen ")]
			}
		}
		return txt
	}())
}

// testNewClasses checks Blackguard, Hexblade, Sorcerer and Wizard, which
// were added to the live toril.db by hand and long missing from init_db.sql:
// who() records new chars of each, and WhoBatch changes a char to one from
// its abbreviation in the who list.
func testNewClasses(t *testing.T, db *sql.DB) {
	for _, n := range []struct{ abbr, class, race, char, alt string }{
		{"Blk", "Blackguard", "Duergar", "Ash", "Ann"},
		{"Hex", "Hexblade", "Human", "Ivy", "Ida"},
		{"Sor", "Sorcerer", "Moon Elf", "Sol", "Sam"},
		{"Wiz", "Wizard", "Gnome", "Wen", "Wes"},
	} {
		// checked first: with foreign keys on, who() can't record a char of a
		// class the table doesn't have, and its error ends the whole test run
		var class string
		err := db.QueryRow("SELECT class_name FROM classes "+
			"WHERE class_abbr = ?", n.abbr).Scan(&class)
		if err != nil || class != n.class {
			t.Errorf("classes check failed: no %s (%s) in init_db.sql",
				n.class, n.abbr)
			continue
		}

		// a new char of the class, as the individual who adds it
		c := Char{name: n.char, lvl: 50, class: n.class + " ", race: n.race,
			acct: "Ash"}
		chk(t, "c.who() "+n.class, []string{}, c.who())
		chkPrefix(t, "char after c.who() "+n.class,
			"t Yog [50 "+n.class+"] "+n.char+" ("+n.race+") (@Ash) seen ",
			ReplyTo("Yog", "char "+n.char))

		// a Warrior the who list now shows in the class
		c = Char{name: n.alt, lvl: 50, class: "Warrior ", race: n.race,
			acct: "Ash"}
		c.who()
		chk(t, "WhoBatch() "+n.class, []string{"who " + n.alt + "\n"},
			WhoBatch(fmt.Sprintf("[50 %s] %s  (%s)", n.abbr, n.alt, n.race)))
		chkPrefix(t, "char after WhoBatch() "+n.class,
			"t Yog [50 "+n.class+"] "+n.alt+" ("+n.race+") (@Ash) seen ",
			ReplyTo("Yog", "char "+n.alt))
	}
}

// testReboot checks what Uptime takes for a reboot: a check whose start,
// when tintin saw it less its uptime, is past the stored boot's by more than
// half that boot's uptime, as with a reboot the bot is around for, one while
// it was offline, and each boot of a crash loop. A long boot isn't one, even
// with the MUD's uptime gaining or losing time, and neither is a check
// written late, in any order, nor an older boot's check written after the
// reboot.
func testReboot(t *testing.T) {
	alerts := 0
	sendBootAlert = func() { alerts++ }
	defer func() { sendBootAlert = SendBootAlert }()

	// a clock that moves only when a case says time has passed, so each
	// check comes as long after the one before as it would have
	clock := now().Truncate(time.Second)
	defer func(was func() time.Time) { now = was }(now)
	now = func() time.Time { return clock }
	wait := func(d time.Duration) { clock = clock.Add(d) }
	// a time check of uptime up that tintin saw ago, before now
	seenAgo := func(up string, ago time.Duration) {
		Uptime(up, clock.Add(-ago))
	}

	db := OpenDB()
	defer db.Close()
	boots := func() int {
		var n int
		chkErr(t, db.QueryRow("SELECT COUNT(*) FROM boots").Scan(&n))
		return n
	}
	// the current boot started ago, and was last seen up for up
	lastBoot := func(ago time.Duration, up string) {
		_, err := db.Exec("UPDATE boots SET boot_time = ?, uptime = ? "+
			"WHERE boot_id = (SELECT MAX(boot_id) FROM boots)",
			clock.Add(-ago).UTC(), up)
		chkErr(t, err)
	}
	// how many boots and alerts there have been by now, and the uptime the
	// current boot was last seen at
	check := func(what string, wantBoots int, wantAlerts int, wantUp string) {
		var up string
		chkErr(t, db.QueryRow("SELECT uptime FROM boots "+
			"ORDER BY boot_id DESC LIMIT 1").Scan(&up))
		if b := boots(); b != wantBoots || alerts != wantAlerts ||
			up != wantUp {
			t.Errorf("Uptime() check failed: %s: Expected: %d boots, "+
				"%d alerts, uptime %s, Actual: %d boots, %d alerts, "+
				"uptime %s", what, wantBoots, wantAlerts, wantUp,
				b, alerts, up)
		}
	}
	// that the current boot is stored as starting ago, before now
	startedAgo := func(what string, ago time.Duration) {
		var start int64
		chkErr(t, db.QueryRow("SELECT CAST(STRFTIME('%s', boot_time) "+
			"AS INTEGER) FROM boots ORDER BY boot_id DESC LIMIT 1").
			Scan(&start))
		if got := clock.Unix() - start; got != int64(ago/time.Second) {
			t.Errorf("Uptime() check failed: %s: Expected: started %v "+
				"ago, Actual: %v ago", what, ago,
				time.Duration(got)*time.Second)
		}
	}
	n := boots()

	// -seen is in Unix seconds, and without it gobot goes by its own clock
	if got := seenAt(1794699709); got.Unix() != 1794699709 {
		t.Errorf("seenAt() check failed: %v", got)
	}
	if got := seenAt(0); !got.Equal(clock) {
		t.Errorf("seenAt(0) check failed: %v, not %v", got, clock)
	}

	// bot offline for 12 hours, MUD still on the same boot
	lastBoot(22*time.Hour, "10:00:00")
	seenAgo("22:00:00", 0)
	check("offline, same boot", n, 0, "22:00:00")

	// MUD rebooted 11 hours ago while the bot was offline: the new uptime
	// is already past the 10 hours the old boot was last seen at
	lastBoot(22*time.Hour, "10:00:00")
	seenAgo("11:00:00", 0)
	check("rebooted while offline", n+1, 1, "11:00:00")
	startedAgo("rebooted while offline", 11*time.Hour)

	// the next check-in is the same boot, and lr starts over with it
	wait(30 * time.Second)
	seenAgo("11:00:30", 0)
	check("next check-in", n+1, 1, "11:00:30")
	chk(t, "lr after reboot",
		[]string{"t Yog No loads reported for current boot.\n"},
		ReplyTo("Yog", "lr"))

	// The MUD lags for four minutes, then answers the checks sent meanwhile
	// all at once, as on 2014-09-20. tintin sees them over a few seconds and
	// runs each in the background, so they're written in any order, here the
	// newest first. Each gives the same start, so the ones behind it, with
	// their lower uptimes, are neither a reboot nor the boot's uptime now.
	wait(4 * time.Minute)
	seenAgo("11:04:30", 0)
	seenAgo("11:04:26", 4*time.Second)
	seenAgo("11:04:27", 3*time.Second)
	seenAgo("11:04:30", 0)
	seenAgo("11:04:29", time.Second)
	seenAgo("11:04:28", 2*time.Second)
	check("checks written late", n+1, 1, "11:04:30")
	startedAgo("checks written late",
		11*time.Hour+4*time.Minute+30*time.Second)

	// A check held up on its way to tintin puts the start late by as much,
	// here four minutes: far less than half the boot's uptime, so it's still
	// the boot's, and a late one.
	wait(30 * time.Second)
	seenAgo("11:01:00", 0)
	check("check seen late", n+1, 1, "11:04:30")

	// a reboot the bot is around for: the uptime drops
	wait(150 * time.Second)
	seenAgo("0:02:00", 0)
	check("uptime dropped", n+2, 2, "0:02:00")

	// a crash loop: a boot last seen 30 seconds in, and a minute later a new
	// one that has been up for 8
	lastBoot(30*time.Second, "0:00:30")
	wait(time.Minute)
	seenAgo("0:00:08", 0)
	check("crash loop", n+3, 3, "0:00:08")

	// In a boot's first minutes too, a check written after a newer one is
	// late, not a reboot: its start is the boot's.
	wait(30 * time.Second)
	seenAgo("0:00:38", 0)
	seenAgo("0:00:08", 30*time.Second)
	check("young boot's check written late", n+3, 3, "0:00:38")

	// A new boot whose first check reaches tintin four minutes late is
	// stored as starting four minutes late, until its next check, on time,
	// puts the start right. A reboot then is caught as after any other boot.
	wait(10 * time.Minute)
	seenAgo("0:00:20", 0) // up 4:20 by now
	check("first check seen late", n+4, 4, "0:00:20")
	wait(30 * time.Second)
	seenAgo("0:04:50", 0)
	check("next check on time", n+4, 4, "0:04:50")
	startedAgo("next check on time", 4*time.Minute+50*time.Second)
	wait(40 * time.Second)
	seenAgo("0:00:20", 0)
	check("reboot after a start put right", n+5, 5, "0:00:20")

	// An uptime that falls by hours is a reboot wherever the times put it:
	// here the old boot is stored as starting half an hour late, as by a
	// clock that was wrong.
	lastBoot(50*time.Hour-30*time.Minute, "50:00:00")
	wait(40 * time.Second)
	seenAgo("0:00:20", 0)
	check("uptime fell by hours", n+6, 6, "0:00:20")

	// the old boot's last check, seen a minute ago, before the new boot
	// began, but written only now: its uptime isn't the new boot's
	seenAgo("50:00:40", time.Minute)
	check("old boot's check written late", n+6, 6, "0:00:20")

	// The MUD's uptime gains or loses a few seconds a day on this server's
	// clock, so after five days, the bot away for the last two and a half,
	// the start a check gives can be half a minute either side of the
	// stored one: still the same boot.
	lastBoot(120*time.Hour, "60:00:00")
	seenAgo("119:59:30", 0)
	check("uptime slow", n+6, 6, "119:59:30")
	startedAgo("uptime slow", 120*time.Hour)
	lastBoot(120*time.Hour, "60:00:00")
	seenAgo("120:00:30", 0)
	check("uptime fast", n+6, 6, "120:00:30")
	startedAgo("uptime fast", 120*time.Hour+30*time.Second)
}

// testSplit checks that long replies go out as tells of at most tellLimit
// characters, each one a whole tell, with nothing lost.
func testSplit(t *testing.T) {
	words := strings.Repeat("word ", 140) // 700 characters
	parts := splitTell(words)
	if len(parts) != 3 || strings.Join(parts, " ") != words {
		t.Errorf("splitTell() check failed: words: %q", parts)
	}
	solid := strings.Repeat("x", 650) // nowhere to break
	parts = splitTell(solid)
	if len(parts) != 3 || strings.Join(parts, "") != solid {
		t.Errorf("splitTell() check failed: no spaces: %d parts", len(parts))
	}
	for _, p := range append(splitTell(words), splitTell(solid)...) {
		if len(p) > tellLimit || p == "" {
			t.Errorf("splitTell() check failed: part of %d chars", len(p))
		}
	}

	// a long load report ahead of a short one: the case that used to come
	// out with a doubled prefix, a blank tell, and a raw line for the MUD
	long := "ogre warlord at the north gate " +
		strings.Repeat("with a very long report ", 14) + "end"
	ReplyTo("Yog", "lr "+long)
	ReplyTo("Bob", "lr short report here")
	txt := ReplyTo("Zed", "lr")
	var text []string
	for _, line := range txt {
		msg := strings.TrimPrefix(line, "t Zed ")
		if msg == line || !strings.HasSuffix(msg, "\n") ||
			len(msg) > tellLimit+1 || strings.TrimSpace(msg) == "" {
			t.Errorf("lr split check failed: bad tell %q", line)
		}
		text = append(text, strings.TrimSuffix(msg, "\n"))
	}
	all := strings.Join(text, " ")
	if len(txt) != 3 ||
		!strings.Contains(all, "1: "+long+" [Yog at ") ||
		!strings.Contains(all, "2: short report here [Bob at ") {
		t.Errorf("lr split check failed: %d tells: %q", len(txt), all)
	}
}

// testTellCap checks that a clist of an account with 60 alts, which runs
// past maxReplyBytes, is cut short and ends with how many more there were,
// and, as nothing narrowed it, how to list fewer.
func testTellCap(t *testing.T, db *sql.DB) {
	_, err := db.Exec("INSERT INTO accounts (account_name) VALUES ('Many')")
	chkErr(t, err)
	alts := 60
	for i := 0; i < alts; i++ {
		_, err = db.Exec("INSERT INTO chars VALUES "+
			"('Many', ?, 'Warrior', 'Human', 50, ?, 't')",
			fmt.Sprintf("Many%c%c", 'a'+i/26, 'a'+i%26), time.Now())
		chkErr(t, err)
	}
	for tell, hint := range map[string]string{
		"clist many": "To list fewer, add a class or level: " +
			"clist many cleric 50",
		// all 60 are level 50 warriors, so this lists as many
		"clist many war 50": "",
	} {
		txt := ReplyTo("Yog", tell)
		bytes := 0
		for _, line := range txt {
			bytes += len(line) + 1
		}
		k := len(txt) - 1
		if bytes > maxReplyBytes || k < 1 ||
			txt[k] != moreTells("Yog", alts-k, hint) {
			t.Errorf("%s cap check failed: %d bytes: %q", tell, bytes, txt)
		}
	}
}

// testHelp checks that each command Katumi takes has a help entry and is
// named in the ? menu, and that help fstat lists the stats, resists and
// slots fstat accepts.
func testHelp(t *testing.T) {
	topics := []string{"?", "hidden?"}
	for _, c := range tellCommands() {
		topics = append(topics, c.name)
	}
	for _, cmd := range topics {
		for _, line := range ReplyTo("Yog", "help "+cmd) {
			if strings.Contains(line, "not found") {
				t.Errorf("help check failed: %s", strings.TrimSpace(line))
			}
		}
	}
	// named the way the menu shows a command: before what follows it, a
	// comma or the end, as in "find <char/acct>", "hidden," and "help
	// fstat)", not as in "<char/acct>"
	shown := strings.Join(menu, "\n")
	for _, c := range tellCommands() {
		named := regexp.MustCompile(`(?m)[ :(]` + regexp.QuoteMeta(c.name) +
			`( <|,|\)|$)`)
		if !named.MatchString(shown) {
			t.Errorf("? menu check failed: no %s in %q", c.name, shown)
		}
	}
	fstat := strings.Join(ReplyTo("Yog", "help fstat"), "")
	for _, want := range []string{"Stats: ac, age, agi, ", " maxagi, ", " mr, ",
		"Resists: acid, ", " fire, ", "Slots: about, ", " ear, ", " on_body, ",
		"Effects: aware, ", " haste, ", "Flags: bless, ", " is_rare, ",
		" two_hand, ", "Usable: evil, ", " fighter, "} {
		if !strings.Contains(fstat, want) {
			t.Errorf("help fstat check failed: no %q in %q", want, fstat)
		}
	}
	// all of it, not cut short to keep the MUD from dropping Katumi
	if strings.Contains(fstat, "too many to send") {
		t.Errorf("help fstat check failed: cut short: %q", fstat)
	}
	astat := strings.Join(ReplyTo("Yog", "help astat"), "")
	if strings.Contains(astat, "buggy") {
		t.Errorf("help astat check failed: %q", astat)
	}
}

// Test_NoHTML runs this test binary from a folder without html/, the way
// the bot runs from its own folder: only the web server should need it.
func Test_NoHTML(t *testing.T) {
	if os.Getenv("GOBOT_NO_HTML") == "1" {
		return // the copy started below: getting here is the test
	}
	exe, err := os.Executable()
	chkErr(t, err)
	cmd := exec.Command(exe, "-test.run=^Test_NoHTML$")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GOBOT_NO_HTML=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("NoHTML check failed: gobot needs html/ outside -port: "+
			"%v\n%s", err, out)
	}
}

// testTimes checks that times are stored in UTC and compared as times, so
// the hour after clocks fall back can't make an old sighting look new, and
// that the one-time fix converts old rows without moving them.
func testTimes(t *testing.T, db *sql.DB, fix string) {
	for _, q := range []string{
		"SELECT CAST(last_seen AS TEXT) FROM chars WHERE char_name = 'Bob'",
		"SELECT CAST(boot_time AS TEXT) FROM boots " +
			"ORDER BY boot_id DESC LIMIT 1",
		"SELECT CAST(report_time AS TEXT) FROM loads " +
			"ORDER BY rowid DESC LIMIT 1",
	} {
		var when string
		chkErr(t, db.QueryRow(q).Scan(&when))
		if !strings.HasSuffix(when, "+00:00") {
			t.Errorf("UTC check failed: %s gave %s", q, when)
		}
	}

	// two alts stored the way the bot used to, in New York time: Dsta 40
	// minutes ago with the offset from before clocks fell back, Dstb 10
	// minutes ago with today's. As text, Dsta's time sorts later.
	loc, err := time.LoadLocation("America/New_York")
	chkErr(t, err)
	_, off := time.Now().In(loc).Zone()
	older := time.Now().Add(-40 * time.Minute).In(time.FixedZone("", off+3600))
	newer := time.Now().Add(-10 * time.Minute).In(time.FixedZone("", off))
	_, err = db.Exec("INSERT INTO accounts (account_name) VALUES ('Dst')")
	chkErr(t, err)
	_, err = db.Exec("INSERT INTO chars VALUES "+
		"('Dst', 'Dsta', 'Cleric', 'Human', 50, ?, 't'), "+
		"('Dst', 'Dstb', 'Warrior', 'Human', 50, ?, 't')", older, newer)
	chkErr(t, err)
	chk(t, "class after fall-back", []string{
		"t Yog No one else online has a level 50 Cleric on the good side.\n"},
		ReplyTo("Yog", "class cleric"))
	if txt := ReplyTo("Yog", "find dst"); len(txt) != 1 ||
		!strings.HasPrefix(txt[0], "t Yog @Dst last seen ") ||
		!strings.HasSuffix(txt[0], " as Dstb\n") {
		t.Errorf("find after fall-back check failed: %q", txt)
	}

	// a report in the oldest format, with no offset: lrdel still finds it
	_, err = db.Exec("INSERT INTO loads VALUES " +
		"((SELECT MAX(boot_id) FROM boots), '2013-03-21 09:35:40', " +
		"'old report here', 'Yog', 'f')")
	chkErr(t, err)
	num := ""
	for _, line := range ReplyTo("Yog", "lr") {
		if n := strings.Index(line, ": old report here [Yog at "+
			"2013-03-21 09:35:40 UTC]"); n > 0 {
			num = strings.TrimPrefix(line[:n], "t Yog ")
		}
	}
	if num == "" {
		t.Errorf("lr check failed: old report not listed")
	}
	chk(t, "lrdel old report", []string{
		"t Yog Load deleted: old report here\n"}, ReplyTo("Yog", "lrdel "+num))
	for _, line := range ReplyTo("Yog", "lr") {
		if strings.Contains(line, "old report here") {
			t.Errorf("lrdel check failed: old report still listed")
		}
	}

	// the one-time fix: every time to UTC without moving it, literal \n in
	// full_stats to real line breaks, and nothing changes a second time
	_, err = db.Exec("UPDATE chars SET last_seen = '2013-04-18 10:31:52.862423' " +
		"WHERE char_name = 'Tom'")
	chkErr(t, err)
	// a toril.db from before TorilDB.db still has its items table, which
	// the fix also cleans up
	_, err = db.Exec("CREATE TABLE items " +
		"(item_id INTEGER PRIMARY KEY, full_stats TEXT)")
	chkErr(t, err)
	_, err = db.Exec("INSERT INTO items (full_stats) VALUES (?)",
		`Name 'x'\nKeyword 'x'\tok`)
	chkErr(t, err)
	instants := func() map[string]float64 {
		m := make(map[string]float64)
		for _, c := range [][2]string{{"chars", "last_seen"},
			{"boots", "boot_time"}, {"loads", "report_time"}} {
			rows, err := db.Query("SELECT rowid, JULIANDAY(" + c[1] +
				") FROM " + c[0])
			chkErr(t, err)
			for rows.Next() {
				var id int64
				var jd float64
				chkErr(t, rows.Scan(&id, &jd))
				m[fmt.Sprintf("%s %d", c[0], id)] = jd
			}
			ChkRows(rows)
		}
		return m
	}
	before := instants()
	script, err := os.ReadFile(fix)
	chkErr(t, err)
	_, err = db.Exec(string(script))
	chkErr(t, err)
	for k, jd := range instants() {
		if diff := (jd - before[k]) * 86400; diff > 0.002 || diff < -0.002 {
			t.Errorf("fix check failed: %s moved %.3fs", k, diff)
		}
	}
	var left, stats int
	chkErr(t, db.QueryRow("SELECT "+
		"(SELECT COUNT(*) FROM chars WHERE last_seen NOT LIKE '%+00:00') + "+
		"(SELECT COUNT(*) FROM boots WHERE boot_time NOT LIKE '%+00:00') + "+
		"(SELECT COUNT(*) FROM loads WHERE report_time NOT LIKE '%+00:00'), "+
		"(SELECT COUNT(*) FROM items WHERE full_stats = ?)",
		"Name 'x'\nKeyword 'x'\tok").Scan(&left, &stats))
	if left != 0 || stats != 1 {
		t.Errorf("fix check failed: %d times not UTC, %d stats fixed",
			left, stats)
	}
	after := instants()
	_, err = db.Exec(string(script))
	chkErr(t, err)
	for k, jd := range instants() {
		if jd != after[k] {
			t.Errorf("fix check failed: second run changed %s", k)
		}
	}
}

// testBackup checks that -bak saves a whole dump, refuses to save one that
// stopped partway or isn't of the bot's database, and prunes old dailies but
// keeps the first of each month. It returns the path of today's backup.
func testBackup(t *testing.T) string {
	chkErr(t, os.MkdirAll("bak", 0750))
	now := time.Now()
	ago := func(days int) string {
		return now.AddDate(0, 0, -days).Format("2006-01-02")
	}
	for _, day := range []string{"2019-12-31", "2020-03-05", "2020-03-12",
		"2020-03-20", "2020-04-15", ago(59), ago(1)} {
		chkErr(t, os.WriteFile(filepath.Join("bak", "toril.db."+day+".gz"),
			[]byte("old"), 0640))
	}
	chkErr(t, os.WriteFile(filepath.Join("bak", "notes.txt"),
		[]byte("keep"), 0640))

	chkErr(t, BackupDB())

	// today's backup is a whole dump, and nothing temporary is left over
	today := filepath.Join("bak", "toril.db."+now.Format("2006-01-02")+".gz")
	f, err := os.Open(today)
	chkErr(t, err)
	zr, err := gzip.NewReader(f)
	chkErr(t, err)
	dump, err := io.ReadAll(zr)
	chkErr(t, err)
	f.Close()
	if !strings.Contains(string(dump), "CREATE TABLE chars") ||
		!strings.HasSuffix(strings.TrimSpace(string(dump)), "COMMIT;") {
		t.Errorf("backup check failed: %d bytes, not a whole dump", len(dump))
	}
	if _, err := os.Stat(today + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("backup check failed: temp file left behind")
	}

	// dailies past 60 days go, the first of each month stays
	for name, keep := range map[string]bool{
		"toril.db.2019-12-31.gz": true, "toril.db.2020-03-05.gz": true,
		"toril.db.2020-03-12.gz": false, "toril.db.2020-03-20.gz": false,
		"toril.db.2020-04-15.gz": true, "toril.db." + ago(59) + ".gz": true,
		"toril.db." + ago(1) + ".gz": true, "notes.txt": true,
	} {
		_, err := os.Stat(filepath.Join("bak", name))
		if kept := err == nil; kept != keep {
			t.Errorf("backup prune check failed: %s kept %v, want %v",
				name, kept, keep)
		}
	}

	// a locked database: the dump fails instead of saving a stub
	lock, err := sql.Open("sqlite3", "toril.db")
	chkErr(t, err)
	defer lock.Close()
	ctx := context.Background()
	conn, err := lock.Conn(ctx)
	chkErr(t, err)
	defer conn.Close()
	_, err = conn.ExecContext(ctx, "BEGIN EXCLUSIVE")
	chkErr(t, err)
	wait := dumpWait
	dumpWait = 100 * time.Millisecond
	if err := writeDump(filepath.Join("bak", "locked.gz")); err == nil {
		t.Errorf("backup check failed: a dump of a locked database passed")
	}
	dumpWait = wait
	_, err = conn.ExecContext(ctx, "ROLLBACK")
	chkErr(t, err)

	// A toril.db that's missing, empty, or the copy dbcopy.sh publishes,
	// which has no online table: -bak fails, makes no toril.db, and leaves
	// the day's earlier backup as it was.
	today, err = filepath.Abs(today)
	chkErr(t, err)
	saved, err := os.ReadFile(today)
	chkErr(t, err)
	here, err := os.Getwd()
	chkErr(t, err)
	defer os.Chdir(here)
	for _, c := range []struct {
		name, want string
		setup      func()
	}{
		{"missing", "toril.db: unable to open database file", func() {}},
		{"empty", "toril.db has no tables", func() {
			chkErr(t, os.WriteFile("toril.db", nil, 0640))
		}},
		{"published", "toril.db has no online table", func() {
			pub, err := sql.Open("sqlite3", "toril.db")
			chkErr(t, err)
			defer pub.Close()
			for _, table := range botTables {
				if table != "online" {
					_, err := pub.Exec("CREATE TABLE " + table + "(x)")
					chkErr(t, err)
				}
			}
		}},
	} {
		chkErr(t, os.Chdir(t.TempDir()))
		day := filepath.Join("bak", filepath.Base(today))
		chkErr(t, os.MkdirAll("bak", 0750))
		chkErr(t, os.WriteFile(day, saved, 0640))
		c.setup()
		if err := BackupDB(); err == nil ||
			!strings.Contains(err.Error(), c.want) {
			t.Errorf("backup check failed: toril.db %s: Expected: %q, "+
				"Actual: %v", c.name, c.want, err)
		}
		if got, err := os.ReadFile(day); err != nil || !bytes.Equal(got, saved) {
			t.Errorf("backup check failed: toril.db %s: the day's backup "+
				"was replaced (%v)", c.name, err)
		}
		if _, err := os.Stat(day + ".tmp"); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("backup check failed: toril.db %s: temp file left "+
				"behind", c.name)
		}
		if c.name != "missing" {
			continue
		}
		// nor does the dump on its own, which opens toril.db read-only
		if err := writeDump(filepath.Join("bak", "dump.gz")); err == nil {
			t.Errorf("backup check failed: toril.db missing: the dump passed")
		}
		if _, err := os.Stat("toril.db"); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("backup check failed: toril.db missing: one was made")
		}
	}
	chkErr(t, os.Chdir(here))
	return today
}

// testRestore checks that -res makes toril.db from a backup, whole, and that
// it neither makes nor changes anything, nor leaves a temp file behind, when
// the backup can't be decompressed, stopped partway, won't load or isn't of
// the bot's database, or when toril.db has tables or left its journal. db is
// the toril.db the backup was made of.
func testRestore(t *testing.T, db *sql.DB, backup string) {
	here, err := os.Getwd()
	chkErr(t, err)
	defer os.Chdir(here)
	saved, err := os.ReadFile(backup)
	chkErr(t, err)
	zr, err := gzip.NewReader(bytes.NewReader(saved))
	chkErr(t, err)
	dump, err := io.ReadAll(zr)
	chkErr(t, err)
	var chars, accounts, boots int
	chkErr(t, db.QueryRow("SELECT (SELECT count(*) FROM chars), "+
		"(SELECT count(*) FROM accounts), (SELECT count(*) FROM boots)").
		Scan(&chars, &accounts, &boots))
	// files lists what's in the working folder, by name, with contents
	files := func() map[string]string {
		entries, err := os.ReadDir(".")
		chkErr(t, err)
		got := make(map[string]string)
		for _, e := range entries {
			b, err := os.ReadFile(e.Name())
			chkErr(t, err)
			got[e.Name()] = string(b)
		}
		return got
	}
	// names lists the names in what files returned, sorted
	names := func(listed map[string]string) []string {
		var got []string
		for name := range listed {
			got = append(got, name)
		}
		slices.Sort(got)
		return got
	}

	// a good backup, where there's no toril.db and where gobot left an
	// empty one while the old one was away: toril.db then dumps exactly as
	// the backup does, and is all that's left in the folder
	for _, empty := range []bool{false, true} {
		chkErr(t, os.Chdir(t.TempDir()))
		if empty {
			chkErr(t, os.WriteFile("toril.db", nil, 0640))
		}
		msg, err := RestoreDB(backup)
		chkErr(t, err)
		chk(t, "RestoreDB()", []string{fmt.Sprintf("Restored toril.db from "+
			"%s: %d chars, %d accounts, %d boots.", backup, chars, accounts,
			boots)}, []string{msg})
		out, err := exec.Command("sqlite3", "toril.db", ".dump").Output()
		chkErr(t, err)
		if !bytes.Equal(out, dump) {
			t.Errorf("restore check failed: toril.db dumps %d bytes, "+
				"the backup %d", len(out), len(dump))
		}
		if left := names(files()); !slices.Equal(left, []string{"toril.db"}) {
			t.Errorf("restore check failed: left behind: %v", left)
		}
	}

	// bad backups
	gz := func(b []byte) []byte {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, err := zw.Write(b)
		chkErr(t, err)
		chkErr(t, zw.Close())
		return buf.Bytes()
	}
	badSum := bytes.Clone(saved)
	badSum[len(badSum)-8] ^= 0xff // the CRC-32 in the gzip trailer
	// how a dump of a locked database ends
	partway := append(bytes.Clone(dump[:bytes.LastIndexByte(
		dump[:len(dump)/2], '\n')+1]), "ROLLBACK; -- due to errors\n"...)
	// the dump -bak saved of a toril.db that wasn't there
	empty := []byte("PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\nCOMMIT;\n")
	// one of the copy dbcopy.sh publishes, which has no online table
	published := []byte("BEGIN TRANSACTION;\n")
	for _, table := range botTables {
		if table != "online" {
			published = fmt.Appendf(published, "CREATE TABLE %s(x);\n", table)
		}
	}
	published = append(published, "COMMIT;\n"...)
	// and one with a syntax error at its first row
	broken := bytes.Replace(dump, []byte("INSERT INTO "),
		[]byte("INSERT INTO INTO "), 1)
	dir := t.TempDir()
	for _, c := range []struct {
		name, want string
		file       []byte // nil for none at all
	}{
		{"missing", "missing.gz", nil},
		{"not gzip", "can't decompress it: gzip: invalid header", dump},
		{"cut short", "can't decompress it: unexpected EOF",
			saved[:len(saved)/2]},
		{"with a bad checksum", "can't decompress it: gzip: invalid checksum",
			badSum},
		{"stopped partway", "doesn't end in COMMIT", gz(partway)},
		{"that won't load", "loading the dump: exit status 1", gz(broken)},
		{"of an empty database", "the backup has no tables", gz(empty)},
		{"of the published copy", "the backup has no online table",
			gz(published)},
	} {
		file := filepath.Join(dir, strings.ReplaceAll(c.name, " ", "-")+".gz")
		if c.file != nil {
			chkErr(t, os.WriteFile(file, c.file, 0640))
		}
		chkErr(t, os.Chdir(t.TempDir()))
		if _, err := RestoreDB(file); err == nil ||
			!strings.Contains(err.Error(), c.want) {
			t.Errorf("restore check failed: a backup %s: Expected: %q, "+
				"Actual: %v", c.name, c.want, err)
		}
		if left := files(); len(left) > 0 {
			t.Errorf("restore check failed: a backup %s: left behind: %v",
				c.name, names(left))
		}
	}

	// toril.db in the way: refused, with nothing changed
	for _, c := range []struct {
		name, want string
		setup      func()
	}{
		{"has tables", "toril.db has tables: move it aside first", func() {
			old, err := sql.Open("sqlite3", "toril.db")
			chkErr(t, err)
			defer old.Close()
			_, err = old.Exec("CREATE TABLE chars(x)")
			chkErr(t, err)
		}},
		{"isn't a database", "toril.db can't be read", func() {
			chkErr(t, os.WriteFile("toril.db", []byte("half a download\n"),
				0640))
		}},
		{"left its journal", "toril.db-journal is left from the old toril.db",
			func() {
				chkErr(t, os.WriteFile("toril.db-journal",
					[]byte("a hot journal\n"), 0640))
			}},
	} {
		chkErr(t, os.Chdir(t.TempDir()))
		c.setup()
		before := files()
		if _, err := RestoreDB(backup); err == nil ||
			!strings.Contains(err.Error(), c.want) {
			t.Errorf("restore check failed: toril.db %s: Expected: %q, "+
				"Actual: %v", c.name, c.want, err)
		}
		if !maps.Equal(files(), before) {
			t.Errorf("restore check failed: toril.db %s: the folder changed",
				c.name)
		}
	}
	chkErr(t, os.Chdir(here))
}

// Test_BotTables checks that botTables, the tables -bak and -res insist on,
// are the ones init_db.sql makes.
func Test_BotTables(t *testing.T) {
	schema, err := os.ReadFile("init_db.sql")
	chkErr(t, err)
	var made []string
	for _, m := range regexp.MustCompile(`(?m)^CREATE TABLE (\w+)`).
		FindAllSubmatch(schema, -1) {
		made = append(made, string(m[1]))
	}
	slices.Sort(made)
	want := slices.Clone(botTables)
	slices.Sort(want)
	if !slices.Equal(made, want) {
		t.Errorf("botTables check failed: init_db.sql makes %v, "+
			"botTables lists %v", made, want)
	}
}

// testRecordsDown checks that a tell needing toril.db gets a reply when the
// file can't be read or written, saying so, rather than no reply or one
// that claims a change that wasn't saved.
func testRecordsDown(t *testing.T, db *sql.DB) {
	down := []string{"t Yog " + recordsDown + "\n"}

	// A long read elsewhere, as a backup's, holds toril.db past dbWait: each
	// write is made, but can't commit. That's what said it was done before.
	var acct, realName sql.NullString
	chkErr(t, db.QueryRow("SELECT a.account_name, a.player_name "+
		"FROM accounts a JOIN chars c ON c.account_name = a.account_name "+
		"WHERE c.char_name = 'Yog'").Scan(&acct, &realName))
	_, err := db.Exec("UPDATE chars SET vis = 't' WHERE char_name = 'Yog'")
	chkErr(t, err)
	_, err = db.Exec("INSERT INTO loads VALUES("+
		"(SELECT MAX(boot_id) FROM boots), ?, 'a test report', 'Yog', 'f')",
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
	chkErr(t, err)
	var reports int
	chkErr(t, db.QueryRow("SELECT count(*) FROM loads WHERE deleted = 'f' "+
		"AND boot_id = (SELECT MAX(boot_id) FROM boots)").Scan(&reports))
	ctx := context.Background()
	reader, err := db.Conn(ctx)
	chkErr(t, err)
	defer reader.Close()
	_, err = reader.ExecContext(ctx, "BEGIN")
	chkErr(t, err)
	var chars int
	chkErr(t, reader.QueryRowContext(ctx, "SELECT count(*) FROM chars").
		Scan(&chars))
	wait := dbWait
	dbWait = 100 * time.Millisecond
	for _, tell := range []string{"delalt Yog", "addalt Yog", "addname Pat",
		"lr a rare at the gate", "lrdel " + strconv.Itoa(reports)} {
		chk(t, "ReplyTo(Yog, "+tell+") while a read holds toril.db", down,
			ReplyTo("Yog", tell))
	}
	dbWait = wait
	_, err = reader.ExecContext(ctx, "ROLLBACK")
	chkErr(t, err)
	// and nothing changed
	var vis, testReport string
	var nowName sql.NullString
	var nowReports int
	chkErr(t, db.QueryRow("SELECT vis FROM chars WHERE char_name = 'Yog'").
		Scan(&vis))
	chkErr(t, db.QueryRow("SELECT player_name FROM accounts "+
		"WHERE account_name = ?", acct).Scan(&nowName))
	chkErr(t, db.QueryRow("SELECT count(*) FROM loads WHERE deleted = 'f' "+
		"AND boot_id = (SELECT MAX(boot_id) FROM boots)").Scan(&nowReports))
	chkErr(t, db.QueryRow("SELECT deleted FROM loads "+
		"WHERE report_text = 'a test report'").Scan(&testReport))
	if vis != "t" || nowName != realName || nowReports != reports ||
		testReport != "f" {
		t.Errorf("failed writes check failed: vis %s, name %v (was %v), "+
			"%d reports (was %d), test report deleted %s", vis, nowName,
			realName, nowReports, reports, testReport)
	}

	// a toril.db that isn't a database: every command that reads it
	here, err := os.Getwd()
	chkErr(t, err)
	chkErr(t, os.Chdir(t.TempDir()))
	defer os.Chdir(here)
	chkErr(t, os.WriteFile("toril.db", []byte("half a download\n"), 0644))
	for _, tell := range []string{"who yog", "clist yog", "char yog",
		"find yog", "name yog", "class cleric", "delalt yog", "addalt yog",
		"addname Pat", "lr", "lr a rare at the gate", "lrdel 1"} {
		chk(t, "ReplyTo(Yog, "+tell+") with toril.db broken", down,
			ReplyTo("Yog", tell))
	}
	// one that doesn't need toril.db still answers as ever
	chkPrefix(t, "ReplyTo(Yog, help lrdel) with toril.db broken",
		"t Yog Syntax: tell katumi lrdel <num>", ReplyTo("Yog", "help lrdel"))
}

// testWhoWaits checks that a who check waits for another process that is
// writing to toril.db, as the -time check katumi.tin starts on the same
// tick may be, instead of failing at once, and that one held up past dbWait
// saves nothing. WhoBatch reads each char's level before updating it, and
// a transaction that has read won't wait for the write lock: it has to
// take the lock first, with BEGIN IMMEDIATE.
func testWhoWaits(t *testing.T, db *sql.DB) {
	const bob = "[50 Bar] Bob - Soul Singer - The Warders (Human)"
	lastSeen := func() string {
		var seen string
		chkErr(t, db.QueryRow("SELECT last_seen FROM chars "+
			"WHERE char_name = 'Bob'").Scan(&seen))
		return seen
	}
	// the other process's write, which commits after hold
	ctx := context.Background()
	writer, err := db.Conn(ctx)
	chkErr(t, err)
	defer writer.Close()
	write := func(hold time.Duration) chan error {
		_, err := writer.ExecContext(ctx, "BEGIN IMMEDIATE")
		chkErr(t, err)
		_, err = writer.ExecContext(ctx, "UPDATE boots SET uptime = uptime")
		chkErr(t, err)
		ended := make(chan error, 1)
		go func() {
			time.Sleep(hold)
			_, err := writer.ExecContext(ctx, "COMMIT")
			ended <- err
		}()
		return ended
	}

	const hold = 300 * time.Millisecond
	before := lastSeen()
	start := time.Now()
	ended := write(hold)
	var txt []string
	failed := catchFatal(func() { txt = WhoBatch(bob) })
	took := time.Since(start)
	chkErr(t, <-ended)
	if failed != "" {
		t.Errorf("WhoBatch() while another process writes: failed after "+
			"%v: %s", took, failed)
	} else {
		chk(t, "WhoBatch() while another process writes", []string{}, txt)
		if took < hold {
			t.Errorf("WhoBatch() while another process writes: done in %v, "+
				"before the write it waits for", took)
		}
		if lastSeen() == before {
			t.Errorf("WhoBatch() while another process writes: " +
				"Bob's last_seen not saved")
		}
	}

	// a write that runs on past dbWait: the check gives up, saving nothing
	before = lastSeen()
	wait := dbWait
	dbWait = 50 * time.Millisecond
	ended = write(hold)
	failed = catchFatal(func() { WhoBatch(bob) })
	dbWait = wait
	chkErr(t, <-ended)
	if failed != "database is locked" {
		t.Errorf("WhoBatch() past dbWait: fatal error %q, "+
			"want \"database is locked\"", failed)
	}
	if lastSeen() != before {
		t.Errorf("WhoBatch() past dbWait: Bob's last_seen saved")
	}
}

// catchFatal runs f, and returns the message of the fatal error it stops
// on, or "" if it doesn't stop on one.
func catchFatal(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(fatalError)
			if !ok {
				panic(r)
			}
			msg = string(e)
		}
	}()
	f()
	return ""
}
