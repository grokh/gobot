package main

import (
	"flag"
)

func main() {
	var acct = flag.String("acct", "", "Character account for initial import.")
	var char = flag.String("char", "", "Character name for DB update or import.")
	var lvl = flag.Int("lvl", 0, "Character level for DB update.")
	var class = flag.String("class", "", "Character class for initial import.")
	var race = flag.String("race", "", "Character race for initial import.")
	var file = flag.String("import", "", "Parse file for identify stats, import to DB.")
	flag.Parse()
	*acct += *class + *race

	if *char != "" && *lvl != 0 {
		Who(*lvl, *char)
	}
	if *file != "" {
		ID(*file)
	}
}
