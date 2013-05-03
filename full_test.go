package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"
)

func chk(t *testing.T, check string, good []string, txt []string) {
	if len(txt) == len(good) {
		for i := range txt {
			if txt[i] != good[i] {
				t.Errorf("%s check failed: Expected: %s, Actual: %s",
					check, good[i], txt[i])
			}
		}
	} else {
		t.Errorf("%s check failed: Incorrect number of responses.", check)
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
	char, lvl, class, race, acct := "Yog", 50, "Shaman    ", "Barbarian", "Yog"
	txt = WhoChar(char, lvl, class, race, acct)
	good = []string{
		"nhc Welcome, Yog. If you have any questions, " +
			"feel free to ask on this channel like this: nhc hi",
	}
	chk(t, "WhoChar()", good, txt)

	char, lvl, class, race, acct = "Bob", 50, "Bard       ", "Human", "Bob"
	txt = WhoChar(char, lvl, class, race, acct)
	good = []string{
		"nhc Welcome, Bob. If you have any questions, " +
			"feel free to ask on this channel like this: nhc hi",
	}
	chk(t, "WhoChar()", good, txt)

	char, lvl, class, race, acct = "Tom", 1, "Warrior     ", "Drow Elf", "Bob"
	txt = WhoChar(char, lvl, class, race, acct)
	good = []string{}
	chk(t, "WhoChar()", good, txt)

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
	good = []string{
		"t Yog I am a Helper Bot (Beta). Valid commands: " +
			"?, help <cmd>, hidden?, who <char>, char <char>, clist <char>, " +
			"find <char>, class <class>, delalt <char>, addalt <char>, " +
			"lr, lr <report>, stat <item>, astat <item>, " +
			"fstat <att> <comp> <val>. " +
			"For further information, tell katumi help <cmd>\n",
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
	good = []string{"t Yog @Bob is online as Tom\n"}
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
			" Dam:4 Hit:5 Haste Slow_Poi " +
			"* (Weapon) Dice:4D4 * Float Magic No_Burn No_Loc !Fighter " +
			"!Mage !Priest * Wt:5 Val:0 * Zone: Unknown * Last ID: " +
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
		"t Yog a black longsword of destruction (Wielded), " +
			"Damroll: 8, Hitroll: 5, " +
			"Fire: 5%, Infravision (Item Type: Weapon) " +
			"Damage Dice: 8D6, Crit Chance: 6%, " +
			"Crit Multiplier: 2x, (Class: Martial, Type: Longsword) * " +
			"Float, Magic, No Burn, No Drop, No Locate, Two Handed " +
			"NO-MAGE ANTI-PALADIN NO-CLERIC ANTI-RANGER\n",
		"t Yog NO-THIEF * Keywords:(black sword destruction twilight) " +
			"* Weight: 15, Value: 10,000 copper * Zone: Unknown * Last ID: " +
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
			"* No_Burn * Wt:0 Val:501,000 * Zone: Unknown * Last ID: " +
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
		"a black longsword of destruction (Wield) " +
			"Dam:8 Hit:5 Fire:5% Infra * (Weapon) Dice:8D6 " +
			"Crit:6% Multi:2x (Class: Martial, Type: Longsword) * " +
			"Float Magic No_Burn No_Drop No_Loc Two_Hand " +
			"!Mage !Pal !Priest !Rang !Thief * Wt:15 Val:10,000 * " +
			"Zone: Unknown * Last ID: " + date + "\n",
		"the mark of the dragonhunter is not in the database.\n",
		"a tiny mithril stud set with a ruby (Ear) " +
			"Dam:3 Maxagi:3 Fire:5% * No_Burn * Wt:0 Val:501,000 " +
			"* Zone: Unknown * Last ID: " + date + "\n",
	}
	chk(t, "GlistStats()", good, txt)

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
