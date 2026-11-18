package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
)

// fatalError is what fatal panics with: an error gobot can't go on from,
// like a toril.db that's locked or broken.
type fatalError string

// fatal stops on an unrecoverable error by panicking with a fatalError. The
// bot runs one command per process, and main ends the run on it as
// log.Fatal would. ReplyTo answers the tell instead, saying Katumi's records
// are unavailable, so the player isn't left with no reply. In the web
// server, exiting would take the whole site down, so eqHandler turns it
// into an HTTP 500 for just the one request.
func fatal(v ...interface{}) {
	panic(fatalError(strings.TrimSuffix(fmt.Sprintln(v...), "\n")))
}

// maxLogEntry is the most bytes of one entry cleanLog writes.
const maxLogEntry = 1000

// cleanLog is where gobot's log goes. Entries repeat what players send in
// tells and what anyone pastes into the site, so each one is kept to a
// single line of at most maxLogEntry bytes, with its control characters
// turned to spaces: a pasted escape code can't act on the terminal the log
// is read in, a carriage return can't dress one entry up as another, and one
// long line can't fill the disk on its own. The log package hands Write one
// whole entry at a time.
type cleanLog struct{ w io.Writer }

func (c cleanLog) Write(p []byte) (int, error) {
	entry := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, strings.ToValidUTF8(strings.TrimRight(string(p), "\r\n"), "?"))
	if len(entry) > maxLogEntry {
		cut := maxLogEntry
		for cut > 0 && !utf8.RuneStart(entry[cut]) {
			cut--
		}
		entry = entry[:cut] + "..."
	}
	if _, err := io.WriteString(c.w, entry+"\n"); err != nil {
		return 0, err
	}
	return len(p), nil
}

func ChkErr(err error) {
	if err != nil {
		fatal(err)
	}
}

// runName names this run by the flags it was given, as "gobot -online", for
// the log entries that end a run: which check a locked toril.db cost, say.
// The values are left out, as one can be a whole who list. flag.Visit gives
// the names in alphabetical order.
func runName() string {
	name := "gobot"
	flag.Visit(func(f *flag.Flag) { name += " -" + f.Name })
	return name
}

// panicSite says where the panic being handled began: gobot's functions from
// there back to main, each with its file and line, as "Stat (items.go:120) <
// ReplyTo (tell.go:1330) < main (gobot.go:160)". Frames above the deepest
// runtime.gopanic are the deferred functions handling the panic, like
// ReplyTo's, which panics again, so they're left out, as is everything
// outside gobot.
func panicSite() string {
	pc := make([]uintptr, 64)
	frames := runtime.CallersFrames(pc[:runtime.Callers(1, pc)])
	// The first frame is this function's, and its name begins the way all
	// of gobot's do: "main." when built, the import path in a test binary.
	f, more := frames.Next()
	pkg := strings.TrimSuffix(f.Function, "panicSite")
	var calls []string
	for more {
		f, more = frames.Next()
		switch {
		case f.Function == "runtime.gopanic":
			calls = nil
		case strings.HasPrefix(f.Function, pkg):
			calls = append(calls, fmt.Sprintf("%s (%s:%d)",
				strings.TrimPrefix(f.Function, pkg),
				filepath.Base(f.File), f.Line))
		}
	}
	return strings.Join(calls, " < ")
}

// endRun is main's last deferred call. A run that panicked with a
// fatalError ends as log.Fatal would end it, logged, under the run's name,
// with exit status 1. Any other panic is a bug: it's logged with where it
// happened, then goes on, to print its stack trace and exit. The trace goes
// to stderr, which tintin leaves on its screen, where nothing keeps it.
func endRun() {
	r := recover()
	if r == nil {
		return
	}
	if msg, ok := r.(fatalError); ok {
		log.Fatalln(runName() + ": " + string(msg))
	}
	log.Printf("Bug in %s: %v, at %s\n", runName(), r, panicSite())
	panic(r)
}

// fail ends a run that couldn't do what it was started for, with exit
// status 1: a backup, a restore, a check of TorilDB.db or an alert. It says
// why in the log and, in the same words, on stderr, for whoever ran it by
// hand and for cron. An error inside any other run goes through fatal
// instead.
func fail(format string, a ...interface{}) {
	why := fmt.Sprintf(format, a...)
	log.Println(why)
	fmt.Fprintln(os.Stderr, why)
	os.Exit(1)
}

func main() {
	f, err := os.OpenFile(
		"logs/bot.log",
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0640,
	)
	if err != nil {
		log.Fatalln(err) // on stderr, with no log to write it in
	}
	defer f.Close()
	log.SetOutput(cleanLog{f})
	// deferred after f.Close, so it runs first, while the log is still open
	defer endRun()

	// for who.go WhoChar(char string, lvl int,
	// class string, race string, acct string)
	var char = flag.String("char", "",
		"Character name for update or import. Ex: Rynshana")
	var lvl = flag.Int("lvl", 0,
		"Character level for update or import. Ex: 50")
	var class = flag.String("class", "",
		"Character class for initial import. Ex: \"Cleric\"")
	var race = flag.String("race", "",
		"Character race for initial import. Ex: \"Moon Elf\"")
	var acct = flag.String("acct", "",
		"Character account for initial import. Ex: Krimic")
	// for who.go WhoBatch(ppl string)
	var who = flag.String("who", "",
		"Batched who output. "+
			"Ex: \"[10 Ctr] Rarac  (Orc)|[ 2 War] Xatus  (Troll)\"")
	// for time.go Uptime(curup string, seen time.Time)
	var time = flag.String("time", "",
		"Parse uptime for boot tracking. Ex: 58:10:26")
	var seen = flag.Int64("seen", 0,
		"When tintin saw the -time line, in Unix seconds, "+
			"or gobot's own clock if left out. Ex: 1794699709")
	// for online.go RecordOnline(n int)
	var online = flag.Int("online", -1,
		"Players online, from who's total, to add to this hour's. Ex: 19")
	// for local.go glstat
	var glist = flag.String("glist", "",
		"Provide stats for multiple items at once. "+
			"Ex: \"a longsword|a dagger\"")
	var item = flag.String("item", "",
		"Provide stats for a single item. Ex: \"a longsword\"")
	// for tell.go ReplyTo(char string, tell string)
	var tell = flag.String("tell", "",
		"Tell with command and maybe operant. Ex: \"stat a longsword\"")
	// run database backup, restore, and parsing
	var backup = flag.Bool("bak", false,
		"Backup the toril.db database.")
	var restore = flag.String("res", "",
		"Restore toril.db from a backup -bak wrote, once the old toril.db "+
			"is moved aside. Ex: bak/toril.db.2026-11-17.gz")
	// for items.go CheckItemDB(path string), run by torildb.sh
	var checkItems = flag.String("checkitems", "",
		"Check a downloaded TorilDB.db before it replaces the one in use. "+
			"Ex: /tmp/TorilDB.db")
	// for ntfy.go SendTestAlert(), to check reboot alerts by hand
	var testAlert = flag.Bool("testalert", false,
		"Send a test alert to the ntfy topic in "+ntfyFile+".")
	// for ntfy.go SendLoginAlert(), which katumi.tin's loginstop runs
	var loginAlert = flag.Bool("loginalert", false,
		"Alert the ntfy topic in "+ntfyFile+" that Katumi has stopped "+
			"logging in, as katumi.tin does once the MUD turns her "+
			"password down twice.")
	// port number for API / web server
	var port = flag.String("port", "", "API / web server port, on 127.0.0.1.")

	// Parse says what's wrong on stderr, which tintin leaves on its screen,
	// so it goes in the log too: a flag this gobot doesn't take loses the
	// event, as when katumi.tin is newer than the gobot installed.
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		log.Printf("Bad flags for %s: %v\n", runName(), err)
		os.Exit(2)
	}

	var cmds []string
	// only run one command at a time
	switch {
	case *time != "":
		Uptime(*time, seenAt(*seen))
	case *online >= 0:
		RecordOnline(*online)
	case *who != "":
		cmds = WhoBatch(*who)
	case *char != "" && *tell != "":
		cmds = ReplyTo(*char, *tell)
	case *char != "" && 50 >= *lvl && *lvl > 0 &&
		*class != "" && *race != "" && *acct != "":
		c := Char{
			name: *char, lvl: *lvl, class: *class, race: *race, acct: *acct,
		}
		cmds = c.who()
	case *item != "":
		for _, line := range FindItem(*item, "short_stats") {
			cmds = append(cmds, line+"\n")
		}
	case *glist != "":
		cmds = GlistStats(*glist)
	case *backup:
		if err := BackupDB(); err != nil {
			fail("Backup failed: %v", err)
		}
	case *restore != "":
		msg, err := RestoreDB(*restore)
		if err != nil {
			fail("Restore failed: %s: %v", *restore, err)
		}
		log.Println(msg)
		cmds = []string{msg + "\n"}
	case *checkItems != "":
		msg, err := CheckItemDB(*checkItems)
		if err != nil {
			fail("%s check failed: %s: %v", itemDBFile, *checkItems, err)
		}
		log.Println(msg)
		cmds = []string{msg + "\n"}
	case *testAlert:
		if err := SendTestAlert(); err != nil {
			fail("Test alert failed: %v", err)
		}
		log.Println("Test alert sent")
		cmds = []string{"Test alert sent to the ntfy topic in " +
			ntfyFile + ".\n"}
	case *loginAlert:
		if err := SendLoginAlert(); err != nil {
			fail("Login alert failed: %v", err)
		}
	case *port != "":
		loadPage()
		http.HandleFunc("/", eqHandler)
		http.Handle("/style.css", http.FileServer(http.Dir("html")))
		http.Handle("/form.js", http.FileServer(http.Dir("html")))
		http.Handle("/favicon.ico", http.FileServer(http.Dir("html")))
		// nginx, on the same machine, is the only way in
		err := http.ListenAndServe("127.0.0.1:"+*port, nil)
		ChkErr(err)
	}
	for _, cmd := range cmds {
		fmt.Print(cmd)
	}
}
