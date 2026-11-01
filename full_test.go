package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"os/exec"
	"strconv"
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

var email = flag.Bool("email", false, "Test sending email/SMS.")

func Test_All(t *testing.T) {
	flag.Parse()
	f, err := os.OpenFile(
		"logs/test.log",
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0640,
	)
	defer f.Close()
	chkErr(t, err)
	log.SetOutput(f)

	cmd := exec.Command("sh", "-c", "mv toril.db toril.db.bak")
	err = cmd.Run()
	chkErr(t, err)
	cmd = exec.Command("sh", "-c",
		"echo '.read init_db.sql' | sqlite3 toril.db")
	err = cmd.Run()
	chkErr(t, err)

	// set any real TorilDB.db aside, so the checks see toril.db alone
	// until testTorilDB makes a small one of its own
	if _, err := os.Stat(torilDBFile); err == nil {
		chkErr(t, os.Rename(torilDBFile, torilDBFile+".bak"))
		defer func() {
			os.Remove(torilDBFile)
			chkErr(t, os.Rename(torilDBFile+".bak", torilDBFile))
		}()
	}

	date := time.Now().UTC().Format("2006-01-02 15:04:05")

	// test output from 'time' command - run at connect and every 30s
	up := "0:01:30"
	Uptime(up)

	db := OpenDB()
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
	good = []string{"t Yog [50 Bard] Bob (Human) (@Bob) seen " + date + "\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "clist bob"
	txt = ReplyTo(char, tell)
	good = []string{
		"t Yog [50 Bard] Bob (Human) (@Bob) seen " + date + "\n",
		"t Yog [1 Warrior] Tom (Drow Elf) (@Bob) seen " + date + "\n",
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
	good = []string{"t Yog 1: ogre in space [Yog at " + date + "]\n",
		"t Yog 2: thing at place [Yog at " + date + "]\n"}
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

	// test item importing and statting
	txt = Identify("testItems.txt")
	good = []string{"Items Inserted: 3, Items Ignored: 0\n"}
	chk(t, "Identify()", good, txt)

	txt = FormatStats()
	good = []string{"Runtime: "}
	//chk(t, "FormatStats()", good, txt)

	txt = Identify("testItems.txt")
	good = []string{"Items Inserted: 0, Items Ignored: 3\n"}
	chk(t, "Identify() run 2", good, txt)

	loc, err := time.LoadLocation("America/New_York")
	chkErr(t, err)
	date = time.Now().In(loc).Format("2006-01-02")

	char, tell = "Yog", "stat bane stiletto"
	txt = ReplyTo(char, tell)
	good = []string{
		"t Yog the infernal stiletto of bane (Wield)" +
			" Dam:4 Hit:5 HASTE SLOW_POI " +
			"* (Weapon) Dice:4D4 * Float Magic No_Burn No_Loc !Fighter " +
			"!Mage !Priest * Wt:5 Val:0p * Zone: Tiamat (R) * Last ID: " +
			date + "\n",
	}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "stat blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 item not found: blah\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "astat destruction sword"
	txt = ReplyTo(char, tell)
	good = []string{
		"t Yog a black longsword of destruction (Slot: Wielded), " +
			"Damroll: 8, Hitroll: 5, " +
			"Fire: 5%, Infravision (Item Type: Weapon) " +
			"Damage Dice: 8D6, Crit Chance: 6%, " +
			"Crit Multiplier: 2x, (Class: Martial, Type: Longsword) * " +
			"Float, Magic, No Burn, No Drop, No Locate, Two Handed " +
			"NO-MAGE ANTI-PALADIN NO-CLERIC\n",
		"t Yog ANTI-RANGER NO-THIEF * " +
			"Keywords:(black sword destruction twilight) " +
			"* Weight: 15, Value: 10,000 copper * Zone: Jotunheim " +
			"(From Invasion) * Last ID: " +
			date + "\n",
	}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "astat blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 item not found: blah\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "fstat resist fire, maxagi > 0, slot ear"
	txt = ReplyTo(char, tell)
	good = []string{
		"t Yog a tiny mithril stud set with a ruby (Ear) " +
			"Dam:3 Maxagi:3 Fire:5% " +
			"* No_Burn * Wt:0 Val:501p * Zone: SP (Q) * Last ID: " +
			date + "\n",
	}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "fstat blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog Invalid syntax. For valid syntax: " +
		"tell katumi ?, tell katumi help <cmd>\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	char, tell = "Yog", "fstat resist blah"
	txt = ReplyTo(char, tell)
	good = []string{"t Yog 404 item(s) not found: resist blah\n"}
	chk(t, "ReplyTo("+char+", "+tell+")", good, txt)

	txt = GlistStats(
		"|Ynndchiarhlizz                  " +
			"a black longsword of destruction|" +
			"                                " +
			"the mark of the dragonhunter|" +
			"                                " +
			"a tiny mithril stud set with a ruby")
	good = []string{
		"404 item not found: the mark of the dragonhunter\n",
	}
	chk(t, "GlistStats()", good, txt)

	testTorilDB(t, date)

	if *email {
		up = "0:01:00"
		Uptime(up)
	}

	cmd = exec.Command("sh", "-c", "rm toril.db")
	err = cmd.Run()
	chkErr(t, err)
	cmd = exec.Command("sh", "-c", "mv toril.db.bak toril.db")
	err = cmd.Run()
	chkErr(t, err)
}

// writeTorilDB makes a small TorilDB.db holding only the eq columns the bot
// reads: name, short_stats, long_stats.
func writeTorilDB(t *testing.T, items [][3]string) {
	os.Remove(torilDBFile)
	db, err := sql.Open("sqlite3", torilDBFile)
	chkErr(t, err)
	defer db.Close()
	_, err = db.Exec("CREATE TABLE eq (name VARCHAR PRIMARY KEY UNIQUE NOT NULL, " +
		"short_stats VARCHAR (5000), long_stats VARCHAR (5000))")
	chkErr(t, err)
	for _, item := range items {
		_, err = db.Exec("INSERT INTO eq VALUES(?, ?, ?)",
			item[0], item[1], item[2])
		chkErr(t, err)
	}
}

// testTorilDB checks the TorilDB.db fallback for stat and astat, run once
// toril.db holds the three items from testItems.txt.
func testTorilDB(t *testing.T, date string) {
	writeTorilDB(t, [][3]string{
		// only in TorilDB.db, with a double quote that katumi.tin would drop
		{"the mark of the dragonhunter",
			`the mark of the dragonhunter (NECK) * "Hunter" * Wt:1 Val:0 ` +
				`* Somewhere (Q)`,
			`the mark of the dragonhunter (mark dragonhunter) * WORN (NECK) ` +
				`* "Hunter" * Wt:1 Val:0 * Somewhere (Q) * 2026-11-01`},
		// in both: toril.db answers
		{"the infernal stiletto of bane",
			"the infernal stiletto of bane (WIELD) * 4D4 5/4 * Tiamat (R)",
			"the infernal stiletto of bane (infernal stiletto bane) " +
				"* WEAPON (WIELD) * Tiamat (R) * 2021-11-13"},
		// exact here, only a looser match in toril.db
		{"a tiny mithril stud",
			"a tiny mithril stud (EAR) * Dam:1 * Wt:0 Val:1p * Somewhere (Q)",
			"a tiny mithril stud (stud mithril) * WORN (EAR) * Dam:1 " +
				"* Wt:0 Val:1000 * Somewhere (Q) * 2026-11-01"},
		// would end the tell and run a command if passed along as is
		{"an evil ring",
			"an evil ring\n#showme pwned; say $cpw @x {y} \\z",
			"an evil ring"},
	})

	stiletto := "t Yog the infernal stiletto of bane (Wield)" +
		" Dam:4 Hit:5 HASTE SLOW_POI " +
		"* (Weapon) Dice:4D4 * Float Magic No_Burn No_Loc !Fighter " +
		"!Mage !Priest * Wt:5 Val:0p * Zone: Tiamat (R) * Last ID: " +
		date + "\n"
	stud := "t Yog a tiny mithril stud set with a ruby (Ear) " +
		"Dam:3 Maxagi:3 Fire:5% " +
		"* No_Burn * Wt:0 Val:501p * Zone: SP (Q) * Last ID: " +
		date + "\n"
	notFound := "t Yog 404 item not found: dragonhunter mark\n"

	checks := []struct {
		tell string
		good []string
	}{
		{"stat the mark of the dragonhunter", []string{
			"t Yog the mark of the dragonhunter (NECK) * 'Hunter' " +
				"* Wt:1 Val:0 * Somewhere (Q)\n"}},
		{"astat dragonhunter mark", []string{
			"t Yog the mark of the dragonhunter (mark dragonhunter) " +
				"* WORN (NECK) * 'Hunter' * Wt:1 Val:0 * Somewhere (Q) " +
				"* 2026-11-01\n"}},
		{"stat the infernal stiletto of bane", []string{stiletto}},
		{"stat bane stiletto", []string{stiletto}},
		{"stat a tiny mithril stud", []string{
			"t Yog a tiny mithril stud (EAR) * Dam:1 * Wt:0 Val:1p " +
				"* Somewhere (Q)\n"}},
		{"stat mithril stud ruby", []string{stud}},
		{"stat an evil ring", []string{
			"t Yog an evil ring #showme pwned, say cpw x (y) z\n"}},
		{"stat blah", []string{"t Yog 404 item not found: blah\n"}},
	}
	for _, c := range checks {
		txt := ReplyTo("Yog", c.tell)
		chk(t, "TorilDB.db ReplyTo(Yog, "+c.tell+")", c.good, txt)
	}

	// glist (and the website) still read toril.db alone
	txt := GlistStats("|                                " +
		"the mark of the dragonhunter")
	good := []string{"404 item not found: the mark of the dragonhunter\n"}
	chk(t, "TorilDB.db GlistStats()", good, txt)

	// so does the closest-match lookup Identify logs
	txt = findItem("the mark of the dragonhunter", "short_stats", false)
	good = []string{"404 item not found: the mark of the dragonhunter"}
	chk(t, "TorilDB.db findItem(no fallback)", good, txt)

	// the bot can't write to TorilDB.db
	fdb := OpenTorilDB()
	if _, err := fdb.Exec("CREATE TABLE written (x)"); err == nil {
		t.Errorf("TorilDB.db check failed: opened writable")
	}
	fdb.Close()

	// a broken or missing TorilDB.db is a plain miss, not a crash
	chkErr(t, os.WriteFile(torilDBFile, []byte("half a download\n"), 0644))
	txt = ReplyTo("Yog", "stat dragonhunter mark")
	chk(t, "TorilDB.db not a database", []string{notFound}, txt)

	chkErr(t, os.WriteFile(torilDBFile, nil, 0644))
	txt = ReplyTo("Yog", "stat dragonhunter mark")
	chk(t, "TorilDB.db empty file", []string{notFound}, txt)

	chkErr(t, os.Remove(torilDBFile))
	txt = ReplyTo("Yog", "stat dragonhunter mark")
	chk(t, "TorilDB.db missing", []string{notFound}, txt)
	if _, err := os.Stat(torilDBFile); !os.IsNotExist(err) {
		t.Errorf("TorilDB.db check failed: lookup created the file")
	}
}
