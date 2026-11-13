package main

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

	// test output from 'time' command - run at connect and every 30s
	up := "0:01:30"
	Uptime(up)

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
		"t Yog Find people: Provide acct and char info: who <char/acct>, " +
			"clist <char/acct>, char <char>, Show last online alt: " +
			"find <char/acct>, Find alts of listed class for people online: " +
			"class <class>, RL names: name <char/acct>, addname <name>, " +
			"Control your listing: delalt <char>, addalt <char>\n",
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

	char, tell = "Yog", "class enchanter"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 class not found: enchanter\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "class bard"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog [50 Bard] Bob (Human) (@Bob)\n"}
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
	testNewClasses(t, db)

	testItems(t, src)
	testReboot(t)
	testOnline(t)
	testSplit(t)
	testTellCap(t, db)
	testTimes(t, db, filepath.Join(src, "fix_times_and_stats.sql"))
	testBackup(t)
}

// chkPrefix checks that a one-line reply starts with want, for replies that
// end in a time that can tick over while the test runs.
func chkPrefix(t *testing.T, check string, want string, txt []string) {
	if len(txt) != 1 || !strings.HasPrefix(txt[0], want) {
		t.Errorf("%s check failed: Expected: %s..., Actual: %q",
			check, strconv.Quote(want), txt)
	}
}

// testCharChanges checks that class and race changes in the game reach the
// DB, starting from Yog as a Barbarian Shaman.
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

// testReboot checks that Uptime notices a reboot even when the bot was
// offline for longer than the old boot had lasted, without mistaking a
// long-running boot for a new one.
func testReboot(t *testing.T) {
	alerts := 0
	sendBootAlert = func() { alerts++ }
	defer func() { sendBootAlert = SendBootAlert }()

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
			time.Now().Add(-ago), up)
		chkErr(t, err)
	}
	check := func(what string, wantBoots int, wantAlerts int) {
		if b := boots(); b != wantBoots || alerts != wantAlerts {
			t.Errorf("Uptime() check failed: %s: Expected: %d boots, "+
				"%d alerts, Actual: %d boots, %d alerts",
				what, wantBoots, wantAlerts, b, alerts)
		}
	}
	n := boots()

	// bot offline for 12 hours, MUD still on the same boot
	lastBoot(22*time.Hour, "10:00:00")
	Uptime("22:00:00")
	check("offline, same boot", n, 0)

	// MUD rebooted 11 hours ago while the bot was offline: the new uptime
	// is already past the 10 hours the old boot was last seen at
	lastBoot(22*time.Hour, "10:00:00")
	Uptime("11:00:00")
	check("rebooted while offline", n+1, 1)

	// the next check-in is the same boot, and lr starts over with it
	Uptime("11:00:30")
	check("next check-in", n+1, 1)
	chk(t, "lr after reboot",
		[]string{"t Yog No loads reported for current boot.\n"},
		ReplyTo("Yog", "lr"))

	// a reboot the bot is around for: the uptime drops
	Uptime("0:02:00")
	check("uptime dropped", n+2, 2)
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
// past maxReplyBytes, is cut short and ends with how many more there were.
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
	txt := ReplyTo("Yog", "clist many")
	bytes := 0
	for _, line := range txt {
		bytes += len(line) + 1
	}
	k := len(txt) - 1
	if bytes > maxReplyBytes || k < 1 || txt[k] != moreTells("Yog", alts-k) {
		t.Errorf("clist cap check failed: %d bytes: %q", bytes, txt)
	}
}

// testHelp checks that each command the ? menu names has a help entry, and
// that help fstat lists the stats, resists and slots fstat accepts.
func testHelp(t *testing.T) {
	for _, cmd := range []string{"?", "help", "hidden", "hidden?", "stat",
		"astat", "fstat", "mstat", "who", "clist", "char", "find", "class",
		"name", "addname", "delalt", "addalt", "lr", "lrdel"} {
		for _, line := range ReplyTo("Yog", "help "+cmd) {
			if strings.Contains(line, "not found") {
				t.Errorf("help check failed: %s", strings.TrimSpace(line))
			}
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
		"t Yog 404 class not found: cleric\n"}, ReplyTo("Yog", "class cleric"))
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
// stopped partway, and prunes old dailies but keeps the first of each month.
func testBackup(t *testing.T) {
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

	BackupDB()

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
}
