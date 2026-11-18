package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// bug panics the way a bug in gobot would, given an n past the end of tells.
func bug(tells []string, n int) string {
	return tells[n]
}

// Test_EndRun checks that a bug's panic goes in the log on one line, with
// where it began, and then goes on. A deferred function that recovers it and
// panics again, as ReplyTo's does, mustn't be taken for where it began.
func Test_EndRun(t *testing.T) {
	var buf bytes.Buffer
	defer log.SetOutput(log.Writer())
	log.SetOutput(cleanLog{&buf})
	for _, again := range []bool{false, true} {
		buf.Reset()
		var after any
		func() {
			defer func() { after = recover() }()
			defer endRun()
			if !again {
				bug([]string{"stat"}, 1)
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						panic(r)
					}
				}()
				bug([]string{"stat"}, 1)
			}()
		}()
		got := buf.String()
		if after == nil {
			t.Errorf("panicked again %v: endRun stopped the panic", again)
		}
		if !strings.Contains(got, " Bug in gobot ") ||
			!strings.Contains(got, ": runtime error: index out of range [1] "+
				"with length 1, at bug (log_test.go:") ||
			strings.Count(got, "\n") != 1 {
			t.Errorf("panicked again %v: endRun logged %q", again, got)
		}
	}
}

// Test_RunLog checks what a run of gobot that ends early leaves in
// logs/bot.log, besides stderr, which tintin leaves on its screen: flags it
// doesn't take, as when katumi.tin is newer than the gobot installed, and a
// fatal error, each with the run's name.
func Test_RunLog(t *testing.T) {
	if testing.Short() {
		t.Skip("builds gobot")
	}
	dir := t.TempDir()
	gobot := filepath.Join(dir, "gobot")
	if runtime.GOOS == "windows" {
		gobot += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", gobot, ".").
		CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	run := filepath.Join(dir, "run")
	chkErr(t, os.MkdirAll(filepath.Join(run, "logs"), 0750))
	for _, c := range []struct {
		args []string
		exit int
		want string
	}{
		{[]string{"-time", "1:02:03", "-nosuch", "x"}, 2,
			"Bad flags for gobot -time: flag provided but not defined: -nosuch"},
		// the toril.db it makes has no tables
		{[]string{"-time", "1:02:03"}, 1, "gobot -time: no such table: boots"},
	} {
		cmd := exec.Command(gobot, c.args...)
		cmd.Dir = run
		err := cmd.Run()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != c.exit {
			t.Errorf("gobot %s: %v, want exit status %d",
				strings.Join(c.args, " "), err, c.exit)
		}
		got := readIfThere(t, filepath.Join(run, "logs", "bot.log"))
		if !strings.Contains(got, c.want+"\n") {
			t.Errorf("gobot %s logged %q, want %q",
				strings.Join(c.args, " "), got, c.want)
		}
	}
}
