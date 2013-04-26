package main

import (
	"testing"
)

func Test_ReplyTo(t *testing.T) {
	txt := ReplyTo("Yog", "blah")
	good := "Invalid syntax. For valid syntax: tell katumi ?, "+
		"tell katumi help <cmd>"
	if txt[0] != good {
		t.Error("'Invalid Syntax' check failed.", txt[0])
	}
	
	txt = ReplyTo("Yog", "?")
	good = "I am a Helper Bot (Beta). Valid commands: "+
		"?, help <cmd>, hidden?, who <char>, char <char>, clist <char>, "+
		"find <char>, class <class>, delalt <char>, addalt <char>, "+
		"lr, lr <report>, stat <item>, astat <item>, fstat <att> <comp> <val>. "+
		"For further information, tell katumi help <cmd>"
	if txt[0] != good {
		t.Error("'Info (?)' check failed.", txt[0])
	}
	
	txt = ReplyTo("Yog", "stat bane stiletto")
	good = "the infernal stiletto of bane (Wield) Dam:4 Hit:5 Haste Slow_Poi "+
		"* (Weapon) Dice:4D4 * Procs: 'Dragonblind' effect: blind, 3 charge - "+
		"'Dragonpoison' effect: poison, 1 charge - 'Dragonslow' effect: slow, 2 charge - "+
		"'Dragonstrike' effect: instant kill, 5 charge * Float Magic No_Burn No_Loc !Fighter "+
		"!Mage !Priest * Wt:5 Val:0 * Zone: Tiamat (R) * Last ID: 2006-01-16"
	if txt[0] != good {
		t.Error("'stat bane stiletto' check failed.", txt[0])
	}
}