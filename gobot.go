package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func ChkErr(err error) {
	if err != nil {
		log.Fatal(err)
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
	// for identify.go Identify(filename string)
	var file = flag.String("import", "",
		"Parse file for identify stats, import to DB. Ex: newstats.txt")
	// for time.go Uptime(curup string)
	var time = flag.String("time", "",
		"Parse uptime for boot tracking. Ex: 58:10:26")
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
	var stats = flag.Bool("s", false,
		"Run FormatStats() creation for item DB.")
	var test = flag.Bool("test", false,
		"Run any current functions under testing.")
	// port number for API / web server
	var port = flag.String("port", "8080", "API / web server port.")

	flag.Parse()

	var cmds []string
	// only run one command at a time
	switch {
	case *time != "":
		Uptime(*time)
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
	case *stats:
		cmds = FormatStats()
	case *item != "":
		cmds = FindItem(*item, "short_stats")
	case *glist != "":
		cmds = GlistStats(*glist)
	case *file != "":
		cmds = Identify(*file)
	case *backup:
		BackupDB()
	case *restore != "":
		RestoreDB(*restore)
	case *test:
		i := Item{}
		i.FillItemByID(1)
		for _, slots := range i.slots {
			for _, slot := range slots {
				fmt.Println(slot)
			}
		}
	case *port != "":
		http.HandleFunc("/torileq/", eqHandler)
		http.HandleFunc("/todrael/", todHandler)
		http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("html/assets"))))
		err := http.ListenAndServe(":"+*port, nil)
		ChkErr(err)
	}
	for _, cmd := range cmds {
		fmt.Print(cmd)
	}
}
