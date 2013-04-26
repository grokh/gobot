package main

import (
	"testing"
	"os"
	"io"
	"bytes"
)

func Test_ReplyTo(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	ReplyTo("Yog", "blah")
	ReplyTo("Yog", "?")
	ReplyTo("Yog", "stat bane stiletto")

	// i don't understand this code
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	w.Close()
	os.Stdout = old
	txt := <-outC

	good := "t Yog Invalid syntax. For valid syntax: tell katumi ?, "+
		"tell katumi help <cmd>\n"
	good += "t Yog I am a Helper Bot (Beta). Valid commands: "+
		"?, help <cmd>, hidden?, who <char>, char <char>, clist <char>, "+
		"find <char>, class <class>, delalt <char>, addalt <char>, "+
		"lr, lr <report>, stat <item>, astat <item>, fstat <att> <comp> <val>. "+
		"For further information, tell katumi help <cmd>\n"
	good += "t Yog the infernal stiletto of bane (Wield) Dam:4 Hit:5 Haste Slow_Poi "+
		"* (Weapon) Dice:4D4 * Procs: 'Dragonblind' effect: blind, 3 charge - "+
		"'Dragonpoison' effect: poison, 1 charge - 'Dragonslow' effect: slow, 2 charge - "+
		"'Dragonstrike' effect: instant kill, 5 charge * Float Magic No_Burn No_Loc !Fighter !M\n"+
		"t Yog age !Priest * Wt:5 Val:0 * Zone: Tiamat (R) * Last ID: 2006-01-16\n"
	if txt != good {
		t.Error("Test failed.")
	}
}