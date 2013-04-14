package main

import (
	"flag"
)

func main() {
	// for who.go Who(char, lvl)
	var char = flag.String("char", "", "Character name for update or import. Ex: Rynshana")
	var lvl = flag.Int("lvl", 0, "Character level for update or import. Ex: 50")
	// for who.go WhoChar(char, lvl, class, race, acct)
	var class = flag.String("class", "", "Character class for initial import. Ex: \"Cleric\"")
	var race = flag.String("race", "", "Character race for initial import. Ex: \"Moon Elf\"")
	var acct = flag.String("acct", "", "Character account for initial import. Ex: Krimic")
	// for identify.go Identify(filename)
	var file = flag.String("import", "", "Parse file for identify stats, import to DB. Ex: newstats.txt")
	// for time.go Uptime(curup)
	var time = flag.String("time", "", "Parse uptime for boot tracking. Ex: 58:10:26")
	// for tell.go
	var cmd = flag.String("cmd", "", "Command from tell. Ex: stat")
	var oper = flag.String("oper", "", "Operant from tell, to be operated on by cmd. Ex: \"a longsword\"")

	flag.Parse()
	*cmd += *oper

	// only run one command at a time
	if *char != "" && *lvl != 0 && *class != "" && *race != "" && *acct != "" {
		WhoChar(*char, *lvl, *class, *race, *acct)
	} else if *char != "" && *lvl != 0 {
		Who(*char, *lvl)
	} else if *file != "" {
		Identify(*file)
	} else if *time != "" {
		Uptime(*time)
	}
}
