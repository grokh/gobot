package main

import (
	"log"
	"os"
	"strings"
	"testing"
)

func chkGood(t *testing.T, char string, tell string, good string, txt []string) {
	if len(txt) == 1 {
		if txt[0] != good {
			t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s",
				char, tell, txt[0])
		}
	} else {
		t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s",
			char, tell, "Incorrect number of responses!")
	}
}

func Test_ReplyTo(t *testing.T) {
	f, err := os.OpenFile("logs/test.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0640)
	defer f.Close()
	ChkErr(err)
	log.SetOutput(f)

	char, tell := "Yog", "blah"
	txt := ReplyTo(char, tell)
	good := "Invalid syntax. For valid syntax: tell katumi ?, " +
		"tell katumi help <cmd>"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "?"
	txt = ReplyTo(char, tell)
	good = "I am a Helper Bot (Beta). Valid commands: " +
		"?, help <cmd>, hidden?, who <char>, char <char>, clist <char>, " +
		"find <char>, class <class>, delalt <char>, addalt <char>, " +
		"lr, lr <report>, stat <item>, astat <item>, fstat <att> <comp> <val>. " +
		"For further information, tell katumi help <cmd>"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "help"
	txt = ReplyTo(char, tell)
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "hidden"
	txt = ReplyTo(char, tell)
	good = "Yog is NOT hidden!"
	chkGood(t, char, tell, good, txt)

	char, tell = "Someone", "hidden"
	txt = ReplyTo(char, tell)
	if len(txt) > 0 {
		t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %v", char, tell, txt)
	}

	char, tell = "Yog", "who blah"
	txt = ReplyTo(char, tell)
	good = "404 character or account not found: blah"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "who bonble"
	txt = ReplyTo(char, tell)
	good = "@Nyyrazzilyss: Nyyrazzilyss, Bonble"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "char Blah"
	txt = ReplyTo(char, tell)
	good = "404 character not found: Blah"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "char bonble"
	txt = ReplyTo(char, tell)
	good = "[5 Bard] Bonble (Halfling) (@Nyyrazzilyss) seen 2013-04-13 10:00:24"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "clist bonble"
	txt = ReplyTo(char, tell)
	good = "[50 Psionicist] Nyyrazzilyss (Illithid) (@Nyyrazzilyss) seen 2013-04-13 09:59:24"
	if len(txt) == 2 {
		if txt[0] != good {
			t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s", char, tell, txt[0])
		}
		good = "[5 Bard] Bonble (Halfling) (@Nyyrazzilyss) seen 2013-04-13 10:00:24"
		if txt[1] != good {
			t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s", char, tell, txt[1])
		}
	} else {
		t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s",
			char, tell, "Incorrect number of responses!")
	}

	char, tell = "Yog", "clist blah"
	txt = ReplyTo(char, tell)
	good = "404 character or account not found: blah"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "lr"
	txt = ReplyTo(char, tell)
	good = "1: ancient brass dragon in DT [Omgiso at 2013-04-13 04:02:33]"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "lr blah"
	txt = ReplyTo(char, tell)
	good = "Invalid syntax. For valid syntax: tell katumi ?, tell katumi help <cmd>"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "lr thing at place"
	txt = ReplyTo(char, tell)
	good = "Load reported: thing at place"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "lr"
	txt = ReplyTo(char, tell)
	good = "2: thing at place [Yog at 20"
	if len(txt) == 2 {
		if !strings.Contains(txt[1], good) {
			t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s", char, tell, txt[1])
		}
	} else {
		t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s",
			char, tell, "Incorrect number of responses!")
	}

	char, tell = "Yog", "lrdel 2"
	txt = ReplyTo(char, tell)
	good = "Load deleted: thing at place"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "lrdel 3"
	txt = ReplyTo(char, tell)
	good = "Invalid load report number."
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "lrdel blah"
	txt = ReplyTo(char, tell)
	good = "Invalid syntax. For valid syntax: tell katumi ?, tell katumi help <cmd>"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "find bonble"
	txt = ReplyTo(char, tell)
	good = "@Nyyrazzilyss last seen"
	if len(txt) == 1 {
		if !strings.Contains(txt[0], good) {
			t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s", char, tell, txt[0])
		}
	} else {
		t.Errorf("ReplyTo Check failed: %s tells you '%s' Actual response: %s",
			char, tell, "Incorrect number of responses!")
	}

	char, tell = "Yog", "find blah"
	txt = ReplyTo(char, tell)
	good = "404 character or account not found: blah"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "delalt bonble"
	txt = ReplyTo(char, tell)
	good = "404 character or account not found: bonble"
	chkGood(t, char, tell, good, txt)

	char, tell = "Bonble", "delalt bonble"
	txt = ReplyTo(char, tell)
	good = "Removed character from your alt list:: bonble"
	chkGood(t, char, tell, good, txt)

	char, tell = "Bonble", "who bonble"
	txt = ReplyTo(char, tell)
	good = "404 character or account not found: bonble"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "addalt bonble"
	txt = ReplyTo(char, tell)
	good = "404 character or account not found: bonble"
	chkGood(t, char, tell, good, txt)

	char, tell = "Bonble", "addalt bonble"
	txt = ReplyTo(char, tell)
	good = "Re-added character to your alt list:: bonble"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "help ?"
	txt = ReplyTo(char, tell)
	good = "Syntax: tell katumi ? -- Katumi provides a full listing of valid commands."
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "help blah"
	txt = ReplyTo(char, tell)
	good = "404 help file not found: blah"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "stat bane stiletto"
	txt = ReplyTo(char, tell)
	good = "the infernal stiletto of bane (Wield) Dam:4 Hit:5 Haste Slow_Poi " +
		"* (Weapon) Dice:4D4 * Procs: 'Dragonblind' effect: blind, 3 charge - " +
		"'Dragonpoison' effect: poison, 1 charge - 'Dragonslow' effect: slow, 2 charge - " +
		"'Dragonstrike' effect: instant kill, 5 charge * Float Magic No_Burn No_Loc !Fighter " +
		"!Mage !Priest * Wt:5 Val:0 * Zone: Tiamat (R) * Last ID: 2006-01-16"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "stat blah"
	txt = ReplyTo(char, tell)
	good = "404 item not found: blah"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "astat destruction sword"
	txt = ReplyTo(char, tell)
	good = "a black longsword of destruction (Wielded), Damroll: 8, Hitroll: 5, " +
		"Fire: 5%, Infravision (Item Type: Weapon) Damage Dice: 8D6 * " +
		"Procs: Battle Rage * Float, Magic, No Burn, No Drop, No Locate, Two Handed " +
		"NO-MAGE ANTI-PALADIN NO-CLERIC ANTI-RANGER NO-THIEF * " +
		"Keywords:(black sword destruction twilight) * Weight: 15, Value: 10,000 copper " +
		"* Zone: Jotunheim (From Invasion) * Last ID: 2008-04-05"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "astat blah"
	txt = ReplyTo(char, tell)
	good = "404 item not found: blah"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "fstat resist fire, maxagi > 0, slot ear"
	txt = ReplyTo(char, tell)
	good = "a tiny mithril stud set with a ruby (Ear) Dam:3 Maxagi:3 Fire:5% " +
		"* No_Burn * Wt:0 Val:501,000 * Zone: SP (Q) * Last ID: 2011-05-12"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "fstat blah"
	txt = ReplyTo(char, tell)
	good = "Invalid syntax. For valid syntax: tell katumi ?, tell katumi help <cmd>"
	chkGood(t, char, tell, good, txt)

	char, tell = "Yog", "fstat resist blah"
	txt = ReplyTo(char, tell)
	good = "404 item(s) not found: resist blah"
	chkGood(t, char, tell, good, txt)
}
