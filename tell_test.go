package main

import (
	"log"
	"os"
	"strings"
	"testing"
)

func Test_ReplyTo(t *testing.T) {
	f, err := os.OpenFile("logs/test.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0640)
	defer f.Close()
	ChkErr(err)
	log.SetOutput(f)

	tell := "blah"
	txt := ReplyTo("Yog", tell)
	good := "Invalid syntax. For valid syntax: tell katumi ?, " +
		"tell katumi help <cmd>"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "?"
	txt = ReplyTo("Yog", tell)
	good = "I am a Helper Bot (Beta). Valid commands: " +
		"?, help <cmd>, hidden?, who <char>, char <char>, clist <char>, " +
		"find <char>, class <class>, delalt <char>, addalt <char>, " +
		"lr, lr <report>, stat <item>, astat <item>, fstat <att> <comp> <val>. " +
		"For further information, tell katumi help <cmd>"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "help"
	txt = ReplyTo("Yog", tell)
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "hidden"
	txt = ReplyTo("Yog", tell)
	good = "Yog is NOT hidden!"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "hidden"
	txt = ReplyTo("Someone", tell)
	if len(txt) > 0 {
		t.Errorf("ReplyTo Check failed: Someone tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "who blah"
	txt = ReplyTo("Yog", tell)
	good = "404 character or account not found: blah"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "who bonble"
	txt = ReplyTo("Yog", tell)
	good = "@Nyyrazzilyss: Nyyrazzilyss, Bonble"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "char Blah"
	txt = ReplyTo("Yog", tell)
	good = "404 character not found: Blah"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "char bonble"
	txt = ReplyTo("Yog", tell)
	good = "[5 Bard] Bonble (Halfling) (@Nyyrazzilyss) seen 2013-04-13 10:00:24"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "clist bonble"
	txt = ReplyTo("Yog", tell)
	good = "[50 Psionicist] Nyyrazzilyss (Illithid) (@Nyyrazzilyss) seen 2013-04-13 09:59:24"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}
	good = "[5 Bard] Bonble (Halfling) (@Nyyrazzilyss) seen 2013-04-13 10:00:24"
	if txt[1] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "clist blah"
	txt = ReplyTo("Yog", tell)
	good = "404 character or account not found: blah"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "lr"
	txt = ReplyTo("Yog", tell)
	good = "1: ancient brass dragon in DT [Omgiso at 2013-04-13 04:02:33]"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "lr blah"
	txt = ReplyTo("Yog", tell)
	good = "Invalid syntax. For valid syntax: tell katumi ?, tell katumi help <cmd>"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "lr thing at place"
	txt = ReplyTo("Yog", tell)
	good = "Load reported: thing at place"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "lr"
	txt = ReplyTo("Yog", tell)
	good = "2: thing at place [Yog at 20"
	if !strings.Contains(txt[1], good) {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "lrdel 2"
	txt = ReplyTo("Yog", tell)
	good = "Load deleted: thing at place"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	/*
	tell = "lrdel 3"
	txt = ReplyTo("Yog", tell)
	good = "Invalid load report number." // haven't actually added this yet
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}
	*/

	tell = "lrdel blah"
	txt = ReplyTo("Yog", tell)
	good = "Invalid syntax. For valid syntax: tell katumi ?, tell katumi help <cmd>"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "find bonble"
	txt = ReplyTo("Yog", tell)
	good = "@Nyyrazzilyss last seen"
	if !strings.Contains(txt[0], good) {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "find blah"
	txt = ReplyTo("Yog", tell)
	good = "404 character or account not found: blah"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "delalt bonble"
	txt = ReplyTo("Yog", tell)
	good = "404 character or account not found: bonble"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "delalt bonble"
	txt = ReplyTo("Bonble", tell)
	good = "Removed character from your alt list:: bonble"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Bonble tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "who bonble"
	txt = ReplyTo("Bonble", tell)
	good = "404 character or account not found: bonble"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Bonble tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "addalt bonble"
	txt = ReplyTo("Yog", tell)
	good = "404 character or account not found: bonble"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "addalt bonble"
	txt = ReplyTo("Bonble", tell)
	good = "Re-added character to your alt list:: bonble"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Bonble tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "help ?"
	txt = ReplyTo("Yog", tell)
	good = "Syntax: tell katumi ? -- Katumi provides a full listing of valid commands."
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "help blah"
	txt = ReplyTo("Yog", tell)
	good = "404 help file not found: blah"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "stat bane stiletto"
	txt = ReplyTo("Yog", tell)
	good = "the infernal stiletto of bane (Wield) Dam:4 Hit:5 Haste Slow_Poi " +
		"* (Weapon) Dice:4D4 * Procs: 'Dragonblind' effect: blind, 3 charge - " +
		"'Dragonpoison' effect: poison, 1 charge - 'Dragonslow' effect: slow, 2 charge - " +
		"'Dragonstrike' effect: instant kill, 5 charge * Float Magic No_Burn No_Loc !Fighter " +
		"!Mage !Priest * Wt:5 Val:0 * Zone: Tiamat (R) * Last ID: 2006-01-16"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "stat blah"
	txt = ReplyTo("Yog", tell)
	good = "404 item not found: blah"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "astat destruction sword"
	txt = ReplyTo("Yog", tell)
	good = "a black longsword of destruction (Wielded), Damroll: 8, Hitroll: 5, "+
		"Fire: 5%, Infravision (Item Type: Weapon) Damage Dice: 8D6 * "+
		"Procs: Battle Rage * Float, Magic, No Burn, No Drop, No Locate, Two Handed "+
		"NO-MAGE ANTI-PALADIN NO-CLERIC ANTI-RANGER NO-THIEF * "+
		"Keywords:(black sword destruction twilight) * Weight: 15, Value: 10,000 copper "+
		"* Zone: Jotunheim (From Invasion) * Last ID: 2008-04-05"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "astat blah"
	txt = ReplyTo("Yog", tell)
	good = "404 item not found: blah"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "fstat resist fire, maxagi > 0, slot ear"
	txt = ReplyTo("Yog", tell)
	good = "a tiny mithril stud set with a ruby (Ear) Dam:3 Maxagi:3 Fire:5% "+
		"* No_Burn * Wt:0 Val:501,000 * Zone: SP (Q) * Last ID: 2011-05-12"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "fstat blah"
	txt = ReplyTo("Yog", tell)
	good = "Invalid syntax. For valid syntax: tell katumi ?, tell katumi help <cmd>"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}

	tell = "fstat resist blah"
	txt = ReplyTo("Yog", tell)
	good = "404 item(s) not found: resist blah"
	if txt[0] != good {
		t.Errorf("ReplyTo Check failed: Yog tells you '%s' Actual response: %s", tell, txt[0])
	}
}
