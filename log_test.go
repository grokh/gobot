package main

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"unicode/utf8"
)

// Test_CleanLog checks that what a player or a visitor to the site sends
// can't reach the log as anything but plain text on the entry's own line.
func Test_CleanLog(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(cleanLog{&buf}, "", 0)

	// an escape code, a carriage return and a line break, as a pasted list
	// or a tell could carry them, then a C1 control and a broken character
	l.Printf("404: item: a \x1b[2Jsword\rFAKE\n2026/01/01 00:00:00 New acct: @x\n")
	l.Printf("404: item: \u009b31m\xff ok")
	want := "404: item: a  [2Jsword FAKE 2026/01/01 00:00:00 New acct: @x\n" +
		"404: item:  31m? ok\n"
	if got := buf.String(); got != want {
		t.Errorf("cleanLog wrote %q, want %q", got, want)
	}

	// one long entry is cut short, on a whole character
	buf.Reset()
	l.Printf("404: item: %s", strings.Repeat("é", maxLogEntry))
	got := buf.String()
	if len(got) > maxLogEntry+len("...\n") || !strings.HasSuffix(got, "...\n") ||
		!utf8.ValidString(got) {
		t.Errorf("long entry: %d bytes, valid %v, ends %q", len(got),
			utf8.ValidString(got), got[len(got)-8:])
	}
}
