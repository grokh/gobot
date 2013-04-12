package main

import (
	"flag"
)

func main() {
        var c = flag.String("who", "", "Check char, add to DB if needed, update last_seen.")
        var i = flag.String("import", "", "Parse file for identify stats, import to DB.")
        flag.Parse()
        Who(*c)
        ID(*i)
}
