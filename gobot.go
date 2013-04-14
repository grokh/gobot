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
	var time = flag.String("time", "", "Parse uptime for boot tracking.")
	var cmd = flag.String("cmd", "", "Command from tell.")
	var oper = flag.String("oper", "", "Operant from tell - to be operated on by cmd.")
	flag.Parse()
	*acct += *class + *race + *time + *cmd + *oper

	if *char != "" && *lvl != 0 {
		Who(*lvl, *char)
	}
	if *file != "" {
		ID(*file)
	}
}
