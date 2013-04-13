package main

import (
	"flag"
)

func main() {
	var char = flag.String("char", "", "Check char, add to DB if needed, update last_seen.")
	var lvl = flag.Int("lvl", 0, "Check char level, update DB.")
	var file = flag.String("import", "", "Parse file for identify stats, import to DB.")
	flag.Parse()

	if *char != "" && *lvl != 0 {
		Who(*lvl, *char)
	}
	if *file != "" {
		ID(*file)
	}
}
