package main

import (
	"testing"
	"log"
	"os"
)

func Test_ReplyTo(t *testing.T) {
	f, err := os.OpenFile("logs/test.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0640)
	defer f.Close()
	ChkErr(err)
	log.SetOutput(f)

	txt := ReplyTo("Yog", "blah")
	good := "Invalid syntax. For valid syntax: tell katumi ?, "+
		"tell katumi help <cmd>"
	if txt[0] != good {
		t.Error("ReplyTo Check failed: Yog tells you 'blah'", txt[0])
	}
	
	txt = ReplyTo("Yog", "?")
	good = "I am a Helper Bot (Beta). Valid commands: "+
		"?, help <cmd>, hidden?, who <char>, char <char>, clist <char>, "+
		"find <char>, class <class>, delalt <char>, addalt <char>, "+
		"lr, lr <report>, stat <item>, astat <item>, fstat <att> <comp> <val>. "+
		"For further information, tell katumi help <cmd>"
	if txt[0] != good {
		t.Error("ReplyTo Check failed: Yog tells you '?'", txt[0])
	}
	
	txt = ReplyTo("Yog", "stat bane stiletto")
	good = "the infernal stiletto of bane (Wield) Dam:4 Hit:5 Haste Slow_Poi "+
		"* (Weapon) Dice:4D4 * Procs: 'Dragonblind' effect: blind, 3 charge - "+
		"'Dragonpoison' effect: poison, 1 charge - 'Dragonslow' effect: slow, 2 charge - "+
		"'Dragonstrike' effect: instant kill, 5 charge * Float Magic No_Burn No_Loc !Fighter "+
		"!Mage !Priest * Wt:5 Val:0 * Zone: Tiamat (R) * Last ID: 2006-01-16"
	if txt[0] != good {
		t.Error("ReplyTo Check failed: Yog tells you 'stat bane stiletto'", txt[0])
	}

	txt = ReplyTo("Yog", "hidden")
	good = "Yog is NOT hidden!"
	if txt[0] != good {
		t.Error("ReplyTo Check failed: Yog tells you 'hidden'", txt[0])
	}

	txt = ReplyTo("Someone", "hidden")
	if len(txt) > 0 {
		t.Error("ReplyTo Check failed: Someone tells you 'hidden'", txt[0])
	}

	txt = ReplyTo("Yog", "who bob")
	good = "404 character or account not found: bob"
	if txt[0] != good {
		t.Error("ReplyTo Check failed: Yog tells you 'who bob'", txt[0])
	}

	txt = ReplyTo("Yog", "who bonble")
	good = "@Nyyrazzilyss: Nyyrazzilyss, Bonble"
	if txt[0] != good {
		t.Error("ReplyTo Check failed: Yog tells you 'who bonble'", txt[0])
	}

	txt = ReplyTo("Yog", "char Bob")
	good = "404 character not found: Bob"
	if txt[0] != good {
		t.Error("ReplyTo Check failed: Yog tells you 'char Bob'", txt[0])
	}

	txt = ReplyTo("Yog", "char bonble")
	good = "[5 Bard] Bonble (Halfling) (@Nyyrazzilyss) seen 2013-04-13 10:00:24"
	if txt[0] != good {
		t.Error("ReplyTo Check failed: Yog tells you 'char bonble'", txt[0])
	}
}
