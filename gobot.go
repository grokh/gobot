package main

import (
	"flag"
)

func main() {
	var acct = flag.String("acct", "", "Character account for initial import.")
	var char = flag.String("char", "", "Check char, add to DB if needed, update last_seen.")
	var lvl = flag.Int("lvl", 0, "Char level for DB update.")
	var class = flag.String("class", "", "Character class for initial import.")
	var race = flag.String("race", "", "Character race for initial import.")
	var file = flag.String("import", "", "Parse file for identify stats, import to DB.")
	flag.Parse()

	if *char != "" && *lvl != 0 {
		Who(*lvl, *char)
	}
	if *file != "" {
		ID(*file)
	}
}
