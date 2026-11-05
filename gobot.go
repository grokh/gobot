package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

// webServer is true when gobot is running as the TorilEQ site (-port).
var webServer bool

// fatal stops on an unrecoverable error. The bot runs one command per
// process, so exiting is fine there. In the web server, exiting would take
// the whole site down, so it panics instead and eqHandler turns that into an
// HTTP 500 for just the one request.
func fatal(v ...interface{}) {
	if webServer {
		panic(fmt.Sprintln(v...))
	}
	log.Fatalln(v...)
}

func ChkErr(err error) {
	if err != nil {
		fatal(err)
	}
}

func main() {
	f, err := os.OpenFile(
		"logs/bot.log",
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0640,
	)
	defer f.Close()
	ChkErr(err)
	log.SetOutput(f)

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
	// for time.go Uptime(curup string)
	var time = flag.String("time", "",
		"Parse uptime for boot tracking. Ex: 58:10:26")
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
		"Restore the toril.db database from backup file. Ex: toril.db.gz")
	// for items.go CheckItemDB(path string), run by torildb.sh
	var checkItems = flag.String("checkitems", "",
		"Check a downloaded TorilDB.db before it replaces the one in use. "+
			"Ex: /tmp/TorilDB.db")
	// port number for API / web server
	var port = flag.String("port", "", "API / web server port, on 127.0.0.1.")

	flag.Parse()

	var cmds []string
	// only run one command at a time
	switch {
	case *time != "":
		Uptime(*time)
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
		BackupDB()
	case *restore != "":
		RestoreDB(*restore)
	case *checkItems != "":
		msg, err := CheckItemDB(*checkItems)
		if err != nil {
			log.Printf("%s check failed: %s: %v\n",
				itemDBFile, *checkItems, err)
			fmt.Fprintf(os.Stderr, "%s check failed: %s: %v\n",
				itemDBFile, *checkItems, err)
			os.Exit(1)
		}
		log.Println(msg)
		cmds = []string{msg + "\n"}
	case *port != "":
		webServer = true
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
