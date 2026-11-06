package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Test_Tintin runs katumi.tin in tintin++ and feeds it fake tells, so the
// real tell trigger, shellsafe and reply handling run against a gobot built
// from this checkout and the test TorilDB.db, through the /bin/sh tintin
// uses. Some of the tells try to get a shell command, a tintin command or a
// variable like $cpw through. Then it logs the script in to a stand-in MUD
// that goes down and drops it, to check that it logs back in (reconnect).
// Set TORILMUD_TIN to the path of the other character's torilmud.tin to run
// it the same way, ASSOC included. The test is skipped without tt++.
//
// It must never reach the MUD. Both scripts log in when tintin starts, so
// for the tells tintin only reads copies without #session lines or login
// events, and for reconnect copies whose only #session line is for the
// stand-in on 127.0.0.1, checked before tt++ runs. The tokens.tin beside
// them has no real account.
func Test_Tintin(t *testing.T) {
	if testing.Short() {
		t.Skip("runs tt++")
	}
	tt := findTintin(t)
	src, err := os.Getwd()
	chkErr(t, err)
	work := t.TempDir()
	bin := filepath.Join(work, "bin")
	chkErr(t, os.MkdirAll(bin, 0750))
	gobot := filepath.Join(bin, "gobot")
	if runtime.GOOS == "windows" {
		gobot += ".exe"
		tt = cygwinRoot(t, tt, work)
	}
	build := exec.Command("go", "build", "-o", gobot, ".")
	build.Dir = src
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	// tt++ -V prints the version, then exits 1
	out, _ := exec.Command(tt, "-V").CombinedOutput()
	if v := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]; v != "" {
		t.Log(v)
	}
	if out, _ := exec.Command(tt, "-h").CombinedOutput(); !strings.Contains(
		string(out), "-H") {
		t.Skip("this tt++ has no -H (nohup) mode, new in 2.02.20, which " +
			"it needs to run without a terminal")
	}

	scripts := []tinScript{
		{name: "katumi.tin", path: filepath.Join(src, "katumi.tin"),
			everyTell: true, online: true,
			login: []string{fakeAcct, fakeSecret, fakeChar}},
	}
	if path := os.Getenv("TORILMUD_TIN"); path != "" {
		// It logs in at the start through the event the server's basic.tin
		// adds, and names its account itself.
		scripts = append(scripts, tinScript{name: "torilmud.tin",
			path: path, deps: []string{"gags.tin", "ynndchiarhlizz.tin"},
			assoc: true, start: "play ynndchiarhlizz;login",
			login: []string{"", fakeSecret, "Ynndchiarhlizz"}})
	}
	for _, s := range scripts {
		t.Run(s.name, func(t *testing.T) {
			t.Run("tells", func(t *testing.T) { s.tells(t, tt, bin, work) })
			t.Run("reconnect", func(t *testing.T) {
				s.reconnect(t, tt, bin, work)
			})
		})
	}
}

// tinScript is a script Test_Tintin runs: the file, others it reads from
// its own folder, whether it answers stat on ASSOC, whether it passes gobot
// every tell, not just stat, and whether it records who's count of players
// online. start is the command that logs in when tintin starts, for a
// script that leaves that to another file. login is what the script sends
// when the MUD asks for an account: the account ("" for any), password and
// character.
type tinScript struct {
	name, path               string
	deps                     []string
	assoc, everyTell, online bool
	start                    string
	login                    []string
}

// checkOnline waits for the count of players online the driver shows, 17,
// to reach the toril.db at path as one who check. gobot runs it in the
// background, so it can land after tt++ is done.
func checkOnline(t *testing.T, path string, screen string) {
	var checks, total int
	for i := 0; i < 50; i++ {
		db, err := sql.Open("sqlite3", path)
		if err == nil {
			err = db.QueryRow("SELECT checks, total FROM online").
				Scan(&checks, &total)
			db.Close()
		}
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if checks != 1 || total != 17 {
		t.Errorf("who's count didn't reach toril.db: %d checks, %d in all"+
			"\nscreen:\n%s", checks, total, screen)
	}
}

// plainDaggers are a dozen items with hit 1, so that fstat hit >= 1 finds
// more items, with the stiletto and the sword, than an fstat reply lists.
func plainDaggers() []dbRow {
	var items []dbRow
	for n := 1; n <= 12; n++ {
		name := fmt.Sprintf("a plain dagger %d", n)
		items = append(items, dbRow{"name": name, "zone": "Waterdeep",
			"type": "WEAPON", "worn": "WIELD", "hit": 1, "date": "2020-01-01",
			"short_stats": name + " (WIELD) * 1D4 1/0", "long_stats": name})
	}
	return items
}

// fakeSecret stands in for the password in $cpw. If it shows up, tintin
// filled in a variable a player typed.
const fakeSecret = "FAKESECRET-91"

// fakeAcct and fakeChar are the account and character in the tokens.tin
// katumi.tin gets.
const (
	fakeAcct = "nobody"
	fakeChar = "Testbot"
)

// writeTokens writes the tokens.tin katumi.tin reads its login from.
func writeTokens(t *testing.T, dir string) {
	chkErr(t, os.WriteFile(filepath.Join(dir, "tokens.tin"), []byte(
		"#var {acct} {"+fakeAcct+"}\n#var {char} {"+fakeChar+"}\n"+
			"#var {cpw} {"+fakeSecret+"}\n"), 0640))
}

// tintinCmd runs tt++ in dir on its test.tin, in nohup mode since there's
// no terminal, with the gobot built for the test first on the PATH.
func tintinCmd(ctx context.Context, tt, bin, dir string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, tt, "-H", "-G", "-r", "test.tin")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"MSYS2_ARG_CONV_EXCL=*") // Git's sh: pass gobot's args as they are
	return cmd
}

// ansiCodes are the escape codes tt++ puts around what it prints.
var ansiCodes = regexp.MustCompile(`\x1b\[[0-9;?>]*[A-Za-z]|\x1b\][^\x07]*\x07`)

func (s tinScript) tells(t *testing.T, tt, bin, work string) {
	run := filepath.Join(work, strings.TrimSuffix(s.name, ".tin"))
	chkErr(t, os.MkdirAll(filepath.Join(run, "logs"), 0750))
	copyOffline(t, s.path, filepath.Join(run, s.name))
	for _, dep := range s.deps {
		copyOffline(t, filepath.Join(filepath.Dir(s.path), dep),
			filepath.Join(run, dep))
	}
	writeItemDB(t, filepath.Join(run, itemDBFile),
		append(append([]dbRow{}, testEq...), plainDaggers()...))
	writeTokens(t, run)
	chkErr(t, os.WriteFile(filepath.Join(run, "test.tin"),
		[]byte(s.driver()), 0640))

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	out, err := tintinCmd(ctx, tt, bin, run).CombinedOutput()
	screen := ansiCodes.ReplaceAllString(
		strings.ReplaceAll(string(out), "\r", ""), "")
	if err != nil {
		t.Fatalf("tt++: %v\n%s", err, screen)
	}
	replies := readIfThere(t, filepath.Join(run, "replies.log"))
	botLog := readIfThere(t, filepath.Join(run, "logs", "bot.log"))

	want := []string{"t Yog " + stilettoShort}
	if s.everyTell {
		// the link to the rest gets through tintin whole
		want = append(want, "t Yog ...and 4 more: "+torileqURL+
			"?fstat=hit+ge+1")
	}
	if s.assoc {
		want = append(want, "acc "+stilettoShort)
	}
	for _, w := range want {
		if !strings.Contains(replies, w+"\n") {
			t.Errorf("no reply %q\nreplies:\n%s\nscreen:\n%s",
				w, replies, screen)
		}
	}
	if s.online {
		checkOnline(t, filepath.Join(run, "toril.db"), screen)
	}
	for _, line := range strings.Split(screen, "\n") {
		switch line = strings.TrimSpace(line); {
		case strings.Contains(line, "#READ"):
			t.Errorf("tintin didn't read the scripts cleanly: %s", line)
		case strings.EqualFold(line, "INJECTED-BRACE") ||
			strings.EqualFold(line, "INJECTED-SEMI"):
			t.Errorf("a tell ran a tintin command: %s", line)
		case strings.HasPrefix(line, "#ERROR") ||
			strings.HasPrefix(line, "#WARNING"):
			t.Log(line)
		}
	}
	// what a shell command or variable from a tell would turn into, had it
	// been run or filled in, in any case: torilmud.tin lowercases tells
	seen := strings.ToLower(screen + replies + botLog)
	for _, bad := range []string{"RAN-42", "912346", fakeSecret} {
		if strings.Contains(seen, strings.ToLower(bad)) {
			t.Errorf("a tell got %q through\nreplies:\n%s\nscreen:\n%s",
				bad, replies, screen)
		}
	}
}

// driver is the test.tin tt++ runs: it loads the script, then fakes the
// lines the MUD would send. Replies go to replies.log instead of the MUD. $
// is made at run time and braces are written \x7B and \x7D, which #showme
// turns into { and }, so the file itself reads cleanly.
func (s tinScript) driver() string {
	lines := []string{
		"#format {dl} {%a} {36}",
		"#var {dir} {.}",
		"#read " + s.name,
		"#var {cpw} {" + fakeSecret + "}",
		"#alias {t} {#line log {replies.log} {t %0}}",
		"#alias {acc} {#line log {replies.log} {acc %0}}",
	}
	for _, text := range []string{
		"stat infernal stiletto",
		"stat ${dl}(echo RAN-${dl}((6*7)))",
		"stat `expr 912345 + 1`",
		"stat ${dl}cpw",
		`stat x\x7D \x7B#showme INJECTED-BRACE\x7D`,
		"stat x;#showme INJECTED-SEMI",
		"fstat hit >= 1",
	} {
		lines = append(lines, "#showme {Yog tells you '"+text+"'}")
		if s.assoc {
			lines = append(lines, "#showme {Yog ASSOC: '"+text+"'}")
		}
	}
	// the last line of a who
	lines = append(lines, "#showme {Total players online: 17.}")
	return strings.Join(append(lines, "#end"), "\n") + "\n"
}

// loginEvent is an #event that logs in when it fires, directly or through a
// #delay nested inside its command.
var loginEvent = regexp.MustCompile(
	`(?i)#event\s*\{[^}]*\}\s*\{.*\b(login|play)\b`)

// connects is a tintin command that can reach the MUD or another host.
var connects = regexp.MustCompile(`(?i)#(ses|ssl|run|port|chat)`)

// copyOffline copies a tintin script without its #session lines (#ses is
// the shortest form tintin takes) or login events, and stops the test if
// anything that could connect is left.
func copyOffline(t *testing.T, from, to string) {
	b, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	var keep []string
	for _, line := range strings.Split(
		strings.ReplaceAll(string(b), "\r", ""), "\n") {
		if !strings.Contains(strings.ToLower(line), "#ses") &&
			!loginEvent.MatchString(line) {
			keep = append(keep, line)
		}
	}
	txt := strings.Join(keep, "\n")
	if found := connects.FindString(txt); found != "" {
		t.Fatalf("%s has %s, which could connect: not running it", from,
			found)
	}
	chkErr(t, os.WriteFile(to, []byte(txt), 0640))
}

func readIfThere(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		t.Error(err)
	}
	return strings.ReplaceAll(string(b), "\r", "")
}

// findTintin finds tt++ on the PATH, or WinTin++ where it installs.
func findTintin(t *testing.T) string {
	if tt, err := exec.LookPath("tt++"); err == nil {
		return tt
	}
	if runtime.GOOS == "windows" {
		tt := filepath.Join(os.Getenv("APPDATA"), "WinTin++", "bin",
			"tt++.exe")
		if _, err := os.Stat(tt); err == nil {
			return tt
		}
	}
	t.Skip("tt++ isn't installed")
	return ""
}

// cygwinRoot readies WinTin++ for #script. It's tt++ built for Cygwin, with
// no shell of its own, so #script and #system fail. A copy of it in a
// Cygwin root of its own, whose /bin is the folder Git for Windows' sh is
// in, gets that sh, and WinTin++ itself is left as it is.
func cygwinRoot(t *testing.T, tt, work string) string {
	dir := filepath.Dir(tt)
	if _, err := os.Stat(filepath.Join(dir, "cygwin1.dll")); err != nil {
		return tt // not WinTin++
	}
	if _, err := os.Stat(filepath.Join(dir, "sh.exe")); err == nil {
		return tt // a full Cygwin, with its own sh
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		sh = `C:\Program Files\Git\usr\bin\sh.exe`
		if _, err := os.Stat(sh); err != nil {
			t.Skip("WinTin++ needs Git for Windows' sh for #script")
		}
	}
	root := filepath.Join(work, "cygwin")
	chkErr(t, os.MkdirAll(filepath.Join(root, "bin"), 0750))
	chkErr(t, os.MkdirAll(filepath.Join(root, "etc"), 0750))
	for _, f := range []string{"tt++.exe", "cygwin1.dll",
		"cygwin-console-helper.exe"} {
		b, err := os.ReadFile(filepath.Join(dir, f))
		if os.IsNotExist(err) && f == "cygwin-console-helper.exe" {
			continue
		}
		chkErr(t, err)
		chkErr(t, os.WriteFile(filepath.Join(root, "bin", f), b, 0750))
	}
	// fstab writes a space as \040
	shDir := strings.ReplaceAll(filepath.ToSlash(filepath.Dir(sh)), " ",
		`\040`)
	chkErr(t, os.WriteFile(filepath.Join(root, "etc", "fstab"),
		[]byte(shDir+" /bin ntfs binary,noacl 0 0\n"), 0640))
	return filepath.Join(root, "bin", "tt++.exe")
}
