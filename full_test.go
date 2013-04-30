package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func chkReply(
	t *testing.T,
	char string,
	tell string,
	good string,
	txt []string,
) {
	if len(txt) == 1 {
		if txt[0] != good {
			t.Errorf("ReplyTo Check failed: %s tells you '%s'"+
				" Actual response: %s",
				char, tell, txt[0])
		}
	} else {
		t.Errorf("ReplyTo Check failed: %s tells you '%s'"+
			" Actual response: %s",
			char, tell, "Incorrect number of responses!")
	}
}

func chkErr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Check failed: %s", err)
	}
}

func Test_All(t *testing.T) {
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
	good := "0:01:30"
	if txt[0] != good {
		t.Errorf("Time check failed: expected %s, got %s.", good, txt[0])
	}

	// test output from 'who' command - run every 30s
	who := "[50 Sha] Yog  (Barbarian)|" +
		"[50 Bar] Bob - Soul Singer - The Warders (Human)|" +
		"[ 1 War] Tom  (Drow Elf)"
	good = "who Yog\nwho Bob\nwho Tom\n"
	txt = WhoBatch(who)
	if strings.Join(txt, "") != good {
		t.Errorf("WhoBatch check failed: expected %s, got %s",
			good, strings.Join(txt, ""))
	}

	// test output from 'who char' command - run whenever new char spotted
	char, lvl, class, race, acct := "Yog", 50, "Shaman    ", "Barbarian", "Yog"
	WhoChar(char, lvl, class, race, acct)
	char, lvl, class, race, acct = "Bob", 50, "Bard       ", "Human", "Bob"
	WhoChar(char, lvl, class, race, acct)
	char, lvl, class, race, acct = "Tom", 1, "Warrior     ", "Drow Elf", "Bob"
	WhoChar(char, lvl, class, race, acct)

	query = "SELECT count(*) FROM chars"
	stmt, err = db.Prepare(query)
	chkErr(t, err)
	defer stmt.Close()

	err = stmt.QueryRow().Scan(&txt[0])
	chkErr(t, err)
	good = "3"
	if txt[0] != good {
		t.Errorf("Who check failed: expected %s, got %s.", good, txt[0])
	}

	// test output from tells
	char, tell := "Yog", "blah"
	txt = ReplyTo(char, tell)
	good = "t Yog Invalid syntax. For valid syntax: tell katumi ?, " +
		"tell katumi help <cmd>\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "?"
	txt = ReplyTo(char, tell)
	good = "t Yog I am a Helper Bot (Beta). Valid commands: " +
		"?, help <cmd>, hidden?, who <char>, char <char>, clist <char>, " +
		"find <char>, class <class>, delalt <char>, addalt <char>, " +
		"lr, lr <report>, stat <item>, astat <item>, fstat <att> <comp> <val>. " +
		"For further information, tell katumi help <cmd>\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "help"
	txt = ReplyTo(char, tell)
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "hidden"
	txt = ReplyTo(char, tell)
	good = "t Yog Yog is NOT hidden!\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Someone", "hidden"
	txt = ReplyTo(char, tell)
	if len(txt) > 0 {
		t.Errorf(
			"ReplyTo Check failed: %s tells you '%s' Actual response: %v",
			char, tell, txt)
	}

	char, tell = "Yog", "who blah"
	txt = ReplyTo(char, tell)
	good = "t Yog 404 character or account not found: blah\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "who bob"
	txt = ReplyTo(char, tell)
	good = "t Yog @Bob: Bob, Tom\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "char Blah"
	txt = ReplyTo(char, tell)
	good = "t Yog 404 character not found: Blah\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "char bob"
	txt = ReplyTo(char, tell)
	good = "t Yog [50 Bard] Bob (Human) (@Bob) seen " + date + "\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "clist bob"
	txt = ReplyTo(char, tell)
	good = "t Yog [50 Bard] Bob (Human) (@Bob) seen " + date + "\n"
	if len(txt) == 2 {
		if txt[0] != good {
			t.Errorf(
				"ReplyTo Check failed: %s tells you '%s' Actual response: %s",
				char, tell, txt[0])
		}
		good = "t Yog [1 Warrior] Tom (Drow Elf) (@Bob) seen " + date + "\n"
		if txt[1] != good {
			t.Errorf(
				"ReplyTo Check failed: %s tells you '%s' Actual response: %s",
				char, tell, txt[1])
		}
	} else {
		t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s",
			char, tell, "Incorrect number of responses!")
	}

	char, tell = "Yog", "clist blah"
	txt = ReplyTo(char, tell)
	good = "t Yog 404 character or account not found: blah\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "lr"
	txt = ReplyTo(char, tell)
	good = "t Yog No loads reported for current boot.\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "lrdel 1"
	txt = ReplyTo(char, tell)
	good = "t Yog No loads reported for current boot.\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "lr blah"
	txt = ReplyTo(char, tell)
	good = "t Yog Invalid syntax. For valid syntax:" +
		" tell katumi ?, tell katumi help <cmd>\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "lr ogre in space"
	txt = ReplyTo(char, tell)
	good = "t Yog Load reported: ogre in space\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "lr thing at place"
	txt = ReplyTo(char, tell)
	good = "t Yog Load reported: thing at place\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "lr"
	txt = ReplyTo(char, tell)
	good = "t Yog 2: thing at place [Yog at " + date + "]\n"
	if len(txt) == 2 {
		if txt[1] != good {
			t.Errorf(
				"ReplyTo Check failed: %s tells you '%s' Actual response: %s",
				char, tell, txt[1])
		}
	} else {
		t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s",
			char, tell, "Incorrect number of responses!")
	}

	char, tell = "Yog", "lrdel 2"
	txt = ReplyTo(char, tell)
	good = "t Yog Load deleted: thing at place\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "lrdel 3"
	txt = ReplyTo(char, tell)
	good = "t Yog Invalid load report number.\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "lrdel blah"
	txt = ReplyTo(char, tell)
	good = "t Yog Invalid syntax. For valid syntax: " +
		"tell katumi ?, tell katumi help <cmd>\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "find bob"
	txt = ReplyTo(char, tell)
	good = "t Yog @Bob is online as Tom\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "find blah"
	txt = ReplyTo(char, tell)
	good = "t Yog 404 character or account not found: blah\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "delalt tom"
	txt = ReplyTo(char, tell)
	good = "t Yog 404 character or account not found: tom\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Bob", "delalt tom"
	txt = ReplyTo(char, tell)
	good = "t Bob Removed character from your alt list:: tom\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Bob", "who tom"
	txt = ReplyTo(char, tell)
	good = "t Bob 404 character or account not found: tom\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "addalt tom"
	txt = ReplyTo(char, tell)
	good = "t Yog 404 character or account not found: tom\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Bob", "addalt tom"
	txt = ReplyTo(char, tell)
	good = "t Bob Re-added character to your alt list:: tom\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "help ?"
	txt = ReplyTo(char, tell)
	good = "t Yog Syntax: tell katumi ? -- " +
		"Katumi provides a full listing of valid commands.\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "help blah"
	txt = ReplyTo(char, tell)
	good = "t Yog 404 help file not found: blah\n"
	chkReply(t, char, tell, good, txt)

	// test item importing and statting
	txt = Identify("testItems.txt")
	good = "Items Inserted: 3, Items Ignored: 0\n"
	if txt[0] != good {
		t.Errorf("Identify() check failed.")
	}
	txt = FormatStats()
	good = "Runtime: "
	if !strings.Contains(txt[0], good) {
		t.Errorf("FormatStats() check failed.")
	}
	txt = Identify("testItems.txt")
	good = "Items Inserted: 0, Items Ignored: 3\n"
	if txt[0] != good {
		t.Errorf("Identify() check #2 failed.")
	}

	loc, err := time.LoadLocation("America/New_York")
	chkErr(t, err)
	date = time.Now().In(loc).Format("2006-01-02")

	char, tell = "Yog", "stat bane stiletto"
	txt = ReplyTo(char, tell)
	good = "t Yog the infernal stiletto of bane (Wield)" +
		" Dam:4 Hit:5 Haste Slow_Poi " +
		"* (Weapon) Dice:4D4 * Float Magic No_Burn No_Loc !Fighter " +
		"!Mage !Priest * Wt:5 Val:0 * Zone: Unknown * Last ID: " + date + "\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "stat blah"
	txt = ReplyTo(char, tell)
	good = "t Yog 404 item not found: blah\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "astat destruction sword"
	txt = ReplyTo(char, tell)
	good = "t Yog a black longsword of destruction (Wielded), " +
		"Damroll: 8, Hitroll: 5, " +
		"Fire: 5%, Infravision (Item Type: Weapon) " +
		"Damage Dice: 8D6, Crit Chance: 6%, " +
		"Crit Multiplier: 2x, (Class: Martial, Type: Longsword) * " +
		"Float, Magic, No Burn, No Drop, No Locate, Two Handed " +
		"NO-MAGE ANTI-PALADIN NO-CLERIC ANTI-RANGER N\n" +
		"t Yog O-THIEF * Keywords:(black sword destruction twilight) * " +
		"Weight: 15, Value: 10,000 copper * Zone: Unknown * Last ID: " +
		date + "\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "astat blah"
	txt = ReplyTo(char, tell)
	good = "t Yog 404 item not found: blah\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "fstat resist fire, maxagi > 0, slot ear"
	txt = ReplyTo(char, tell)
	good = "t Yog a tiny mithril stud set with a ruby (Ear) " +
		"Dam:3 Maxagi:3 Fire:5% " +
		"* No_Burn * Wt:0 Val:501,000 * Zone: Unknown * Last ID: " +
		date + "\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "fstat blah"
	txt = ReplyTo(char, tell)
	good = "t Yog Invalid syntax. For valid syntax: " +
		"tell katumi ?, tell katumi help <cmd>\n"
	chkReply(t, char, tell, good, txt)

	char, tell = "Yog", "fstat resist blah"
	txt = ReplyTo(char, tell)
	good = "t Yog 404 item(s) not found: resist blah\n"
	chkReply(t, char, tell, good, txt)

	txt = GlistStats(
		"|Ynndchiarhlizz                  " +
		"a black longsword of destruction|" +
		"                                " +
		"the mark of the dragonhunter|" +
		"                                " +
		"a tiny mithril stud set with a ruby")
	good = "a black longsword of destruction (Wield) " +
		"Dam:8 Hit:5 Fire:5% Infra * (Weapon) Dice:8D6 " +
		"Crit:6% Multi:2x (Class: Martial, Type: Longsword) * " +
		"Float Magic No_Burn No_Drop No_Loc Two_Hand " +
		"!Mage !Pal !Priest !Rang !Thief * Wt:15 Val:10,000 * " +
		"Zone: Unknown * Last ID: " + date + "\n" +
		"the mark of the dragonhunter is not in the database.\n" +
		"a tiny mithril stud set with a ruby (Ear) " +
		"Dam:3 Maxagi:3 Fire:5% * No_Burn * Wt:0 Val:501,000 " +
		"* Zone: Unknown * Last ID: "+date+"\n"
	if strings.Join(txt, "") != good {
		t.Errorf("GlistStats() check failed.")
	}

	cmd = exec.Command("sh", "-c", "rm toril.db")
	err = cmd.Run()
	chkErr(t, err)
	cmd = exec.Command("sh", "-c", "mv toril.db.bak toril.db")
	err = cmd.Run()
	chkErr(t, err)
}
