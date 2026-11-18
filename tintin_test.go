package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
// that goes down, garbles a line, drops it and goes quiet, to check that it
// answers each login prompt in turn and logs back in (reconnect), and to
// one that turns down its password, to check that it stops (password).
// Set TORILMUD_TIN to the path of the other character's torilmud.tin to run
// it the same way, ASSOC included. The test is skipped without tt++.
//
// It must never reach the MUD. Both scripts log in when tintin starts, so
// for the tells tintin only reads copies without #session lines or login
// events, and for the logins copies whose only #session line is for the
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
			everyTell: true, online: true, uptime: true, daily: true,
			chars: true, botLog: true, stamped: true,
			login: []string{fakeAcct, fakeSecret, fakeChar}, tick: "who"},
	}
	if path := os.Getenv("TORILMUD_TIN"); path != "" {
		// It logs in at the start through the event the server's basic.tin
		// adds, and names its account itself.
		scripts = append(scripts, tinScript{name: "torilmud.tin",
			path: path, deps: []string{"gags.tin", "ynndchiarhlizz.tin"},
			assoc: true, start: "play ynndchiarhlizz;login",
			login: []string{"", fakeSecret, "Ynndchiarhlizz"}, tick: "l"})
	}
	for _, s := range scripts {
		t.Run(s.name, func(t *testing.T) {
			t.Run("tells", func(t *testing.T) { s.tells(t, tt, bin, work) })
			t.Run("reconnect", func(t *testing.T) {
				s.reconnect(t, tt, bin, work)
			})
			t.Run("password", func(t *testing.T) {
				s.password(t, tt, bin, work)
			})
		})
	}
}

// tinScript is a script Test_Tintin runs: the file, others it reads from
// its own folder, whether it answers stat on ASSOC, whether it passes gobot
// every tell, not just stat, whether it records who's count of players
// online, whether it records the MUD's uptime, whether it backs up
// toril.db at midnight (an #event {DAY}), and whether it adds the chars
// that who <name> lines show. botLog is whether it notes in
// logs/bot.log when it starts, connects, logs in, drops and quits, and the
// logins that go wrong, and sends an alert when it stops logging in, and
// stamped whether its session log has the time each line came. start is the
// command that logs in when tintin starts, for a script that leaves that to
// another file. login is what the script answers the MUD's login prompts
// with: the account ("" for any), the password and the character. tick is a
// line its ticker sends once the character is in.
type tinScript struct {
	name, path                              string
	deps                                    []string
	assoc, everyTell, online, uptime, daily bool
	chars, botLog, stamped                  bool
	start, tick                             string
	login                                   []string
}

// driverUptime is the uptime the driver shows, as a time check's answer.
const driverUptime = "58:10:26"

// checkUptime waits for driverUptime to reach the toril.db at path as the
// current boot's, and checks the boot's start: when tintin saw the line,
// sometime from from to to, less the uptime. It has to be a whole second,
// as only -seen gives it: by gobot's own clock it would have a fraction.
// gobot runs in the background, so it can land after tt++ is done.
func checkUptime(t *testing.T, path, screen string, from, to time.Time) {
	var up, start string
	for i := 0; i < 50; i++ {
		db, err := sql.Open("sqlite3", path)
		if err == nil {
			err = db.QueryRow("SELECT uptime, CAST(boot_time AS TEXT) "+
				"FROM boots ORDER BY boot_id DESC LIMIT 1").Scan(&up, &start)
			db.Close()
		}
		if err == nil && up == driverUptime {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	d := parseUptime(driverUptime)
	// Parse takes a fraction of a second the layout doesn't have
	when, err := time.Parse("2006-01-02 15:04:05-07:00", start)
	if up != driverUptime || err != nil || when.Nanosecond() != 0 ||
		when.Before(from.Truncate(time.Second).Add(-d)) ||
		when.After(to.Add(-d)) {
		t.Errorf("the uptime didn't reach toril.db timed by -seen: uptime "+
			"%q, start %q, seen from %v to %v\nscreen:\n%s", up, start,
			from.UTC(), to.UTC(), screen)
	}
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

// whoLines are lines the driver shows as who <name>'s answers, which the
// who trigger passes to gobot, and what toril.db should then hold of each
// char: its level, class, name, race and account.
var whoLines = []struct{ line, char string }{
	// a level of one digit, on an account new to the DB
	{"[ 1 Wizard    ] Newbie  (Yuan-Ti) (@Newacct)",
		"1 Wizard Newbie Yuan-Ti Newacct"},
	// something between the race and the account
	{"[50 Enchanter ] Oldhand  (Moon Elf) (RP) (@Oldacct)",
		"50 Enchanter Oldhand Moon Elf Oldacct"},
	// a light flag after the account
	{"[50 Anti-Paladin] Shade  (Drow Elf) (@Shadeacct) (Daylight)",
		"50 Anti-Paladin Shade Drow Elf Shadeacct"},
	// an account written into the line ahead of the MUD's own: the last
	// one on the line is the one that counts
	{"[50 Warrior   ] Sneak (Human) (@Victim)  (Human) (@Realacct)",
		"50 Warrior Sneak Human Realacct"},
}

// checkChars checks that the who trigger gave gobot each of whoLines whole:
// the toril.db at path has their chars, each under the account the MUD
// showed, and no account but those, and the newcomer was welcomed. The
// trigger waits for gobot, so they're all in by the time tt++ is done.
func checkChars(t *testing.T, path, replies, screen string) {
	db, err := sql.Open("sqlite3", path)
	chkErr(t, err)
	defer db.Close()
	var chars, accts []string
	for _, w := range whoLines {
		chars = append(chars, w.char)
		accts = append(accts, w.char[strings.LastIndex(w.char, " ")+1:])
	}
	if got := queryStrings(t, db, "SELECT char_level || ' ' || class_name "+
		"|| ' ' || char_name || ' ' || char_race || ' ' || account_name "+
		"FROM chars ORDER BY rowid"); !reflect.DeepEqual(got, chars) {
		t.Errorf("who lines gave toril.db the chars %q, want %q\nscreen:\n%s",
			got, chars, screen)
	}
	if got := queryStrings(t, db, "SELECT account_name FROM accounts "+
		"ORDER BY rowid"); !reflect.DeepEqual(got, accts) {
		t.Errorf("who lines gave toril.db the accounts %q, want %q", got, accts)
	}
	if welcome := "nhc Welcome, Newbie. If you have any questions, feel " +
		"free to ask on this channel.\n"; !strings.Contains(replies, welcome) {
		t.Errorf("no welcome for the newcomer\nreplies:\n%s", replies)
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
	if s.uptime {
		// the bot's own tables, as the uptime goes in its current boot
		schema, err := os.ReadFile("init_db.sql")
		chkErr(t, err)
		db, err := sql.Open("sqlite3", filepath.Join(run, "toril.db"))
		chkErr(t, err)
		_, err = db.Exec(string(schema))
		chkErr(t, err)
		chkErr(t, db.Close())
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	from := time.Now()
	out, err := tintinCmd(ctx, tt, bin, run).CombinedOutput()
	to := time.Now()
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
	// The tells quoting another trigger's words went to the tell trigger:
	// Katumi answers both as tells, and only the ASSOC line itself gets an
	// answer on ASSOC.
	if n := strings.Count(replies, "t Yog "+BadSyntax+"\n"); s.everyTell &&
		n != 2 {
		t.Errorf("%d tells got %q, want 2: a tell quoting another "+
			"trigger's words went to that trigger\nreplies:\n%s",
			n, BadSyntax, replies)
	}
	if n := strings.Count(replies, "acc "+stilettoShort+"\n"); s.assoc &&
		n != 1 {
		t.Errorf("%d answers on ASSOC for the stiletto, want 1: a tell "+
			"quoting an ASSOC line went to the ASSOC trigger\nreplies:\n%s",
			n, replies)
	}
	// A tell gets every plain dagger, in order. ASSOC gets only the closest,
	// as torilmud.tin passes on just the last line gobot -item prints.
	var daggers string
	for n := 1; n <= itemLimit; n++ {
		daggers += fmt.Sprintf("t Yog a plain dagger %d (WIELD) * 1D4 1/0\n", n)
	}
	daggers += "t Yog ...and 2 more items by that name.\n"
	if !strings.Contains(replies, daggers) {
		t.Errorf("no tells of the plain daggers, in order:\n%s\nreplies:\n%s",
			daggers, replies)
	}
	if n := strings.Count(replies, "acc a plain dagger"); s.assoc && (n != 1 ||
		!strings.Contains(replies, "acc a plain dagger 1 (WIELD) * 1D4 1/0\n")) {
		t.Errorf("%d answers on ASSOC for the plain daggers, want 1, the "+
			"first\nreplies:\n%s", n, replies)
	}
	if s.online {
		checkOnline(t, filepath.Join(run, "toril.db"), screen)
	}
	if s.uptime {
		checkUptime(t, filepath.Join(run, "toril.db"), screen, from, to)
	}
	if s.chars {
		checkChars(t, filepath.Join(run, "toril.db"), replies, screen)
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
		"#alias {nhc} {#line log {replies.log} {nhc %0}}",
	}
	for _, text := range []string{
		"stat infernal stiletto",
		"stat plain dagger",
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
	// Tells quoting another trigger's words. tintin fires only the first
	// action a line matches, so these have to reach the tell trigger first.
	lines = append(lines,
		"#showme {Yog tells you 'That character doesn't exist! "+
			"Please try again.'}",
		"#showme {Yog tells you 'x ASSOC: 'stat infernal stiletto'}")
	// who <name>'s answers, for a script that adds chars from them
	if s.chars {
		for _, w := range whoLines {
			lines = append(lines, "#showme {"+w.line+"}")
		}
	}
	// the answer to a time check, then the last line of a who
	lines = append(lines,
		"#showme {Time elapsed since boot-up: "+driverUptime+"}",
		"#showme {Total players online: 17.}")
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
