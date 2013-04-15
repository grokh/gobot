package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
	//"time"
)

func NotFound(four string, oper string) string {
	return "404 " + four + " not found: " + oper
}

func Reply(char string, msg string) {
	fmt.Printf("t %s %s\n", char, msg)
}

func FindItem(item string, length string) string {
	return item + length
}

func ReplyTo(char string, tell string) {
	info := "I am a Helper Bot (Beta). " +
		"Valid commands: ?, help <cmd>, hidden?, who <char>, char <char>, " +
		"clist <char>, find <char>, class <class>, delalt <char>, addalt <char>, " +
		"lr, lr <report>, stat <item>, astat <item>, fstat <att> <comp> <val>. " +
		"For further information, tell katumi help <cmd>"
	// By default, replies will use 'invalid syntax', requiring reassignment
	syntax := "Invalid syntax. For valid syntax: tell katumi ?, " +
		"tell katumi help <cmd>"
	txt := syntax

	oper := ""
	split := strings.SplitN(tell, " ", 2)
	cmd := strings.ToLower(split[0])
	if len(split) > 1 {
		oper = split[1]
	}

	// debugging
	fmt.Printf("Cmd : %s\n", cmd)
	fmt.Printf("Oper: %s\n", oper)

	switch {
	case cmd == "?":
		Reply(char, info)
	case cmd == "help":
		switch {
		case oper == "":
			Reply(char, info)
		case oper == "?":
			txt = "Syntax: tell katumi ? -- " +
				"Katumi provides a full listing of valid commands."
			Reply(char, txt)
		case oper == "hidden?":
			txt = "Syntax: tell katumi hidden? -- " +
				"Katumi sends a tell in reply " +
				"if she can see you. If you receive no reply, you are hidden. " +
				"Katumi has permanent detect invis to ensure that won't cause " +
				"issues."
			Reply(char, txt)
		case oper == "who":
			txt = "Syntax: tell katumi who <acct/char> -- " +
				"Example: tell katumi who rynshana -- " +
				"Katumi provides the account name along with a list of every " +
				"known alt of the named character as a reply. Also works with " +
				"account names."
			Reply(char, txt)
		case oper == "char":
			txt = "Syntax: tell katumi char <char> -- " +
				"Example: tell katumi char rynshana -- " +
				"Katumi provides the account name along with full information " +
				"on the named character as a reply, " +
				"to include level, class, race, and date/time last seen."
			Reply(char, txt)
		case oper == "find":
			txt = "Syntax: tell katumi find <acct/char> -- " +
				"Example: tell katumi find rynshana -- " +
				"Katumi provides the account name along with the last known " +
				"sighting of any of that character's alts. If they have an alt online, " +
				"the time will measure in seconds. Also works with account names."
			Reply(char, txt)
		case oper == "clist":
			txt = "Syntax: tell katumi clist <acct/char> -- " +
				"Example: tell katumi clist rynshana -- " +
				"Katumi provides a full " +
				"listing of every known alt belonging to <char>, including race, " +
				"class, level, and date/time last seen, matching the format of " +
				"the 'char' command. Also works with account names."
			Reply(char, txt)
		case oper == "class":
			txt = "Syntax: tell katumi class <class> -- " +
				"Example: tell katumi class enchanter -- " +
				"Katumi provides a " +
				"list of alts matching the named class for characters who " +
				"are currently online, letting group leaders find useful " +
				"alts from the 'who' list."
			Reply(char, txt)
		case oper == "delalt":
			txt = "Syntax: tell katumi delalt <char> -- " +
				"Example: tell katumi delalt rynshana -- " +
				"Katumi no longer " +
				"provides information on the alt, removing it from 'clist', " +
				"'who', and 'find' commands. Only works for characters " +
				"attached to the same account requesting the removal."
			Reply(char, txt)
		case oper == "addalt":
			txt = "Syntax: tell katumi addalt <char> -- " +
				"Example: tell katumi addalt rynshana -- " +
				"Katumi begins " +
				"providing information on the named alt, who had previously " +
				"been removed with 'delalt', adding the character back to " +
				"'clist', 'who', and 'find' commands. Only works for chars " +
				"attached to the same account."
			Reply(char, txt)
		case oper == "lr":
			txt = "Syntax: tell katumi lr -- " +
				"Katumi provides a list of load " +
				"reports for the current boot. This could be rares or quests " +
				"other players have found or completed. Use the 'lrdel' command " +
				"to remove bad or out of date reports."
			Reply(char, txt)
			txt = "Syntax: tell katumi lr <report> -- " +
				"Example: tell katumi lr timestop gnome at ako village -- " +
				"Katumi adds <report> " +
				"to the list of load reports for the current boot. If you find " +
				"a rare, global load, or complete a quest or the like, report " +
				"it along with a location so other players will know!"
			Reply(char, txt)
		case oper == "lrdel":
			txt = "Syntax: tell katumi lrdel <num> -- " +
				"Example: tell katumi lrdel 3 -- " +
				"Katumi removes the " +
				"numbered item from the load reports, if a quest is completed " +
				"or a rare killed, or a report found to be inaccurate. Please " +
				"do not abuse this command - this service helps everyone."
			Reply(char, txt)
		case oper == "stat":
			txt = "Syntax: tell katumi stat <item> -- " +
				"Example: tell katumi stat isha cloak -- " +
				"Katumi provides stat info for the item named. " +
				"Use 'astat' for full text of acronyms and keywords. " +
				"The name search is fairly forgiving. Please send new stats " +
				"in an mwrite to katumi or email to kristi.michaels@gmail.com"
			Reply(char, txt)
		case oper == "astat":
			txt = "Syntax: tell katumi astat <item> -- " +
				"Example tell katumi astat destruction sword -- " +
				"Katumi provides full " +
				"stat information for the item named. Use 'stat' for short " +
				"text. The name search is fairly forgiving, though the stats " +
				"are a little buggy right now since I haven't put much time " +
				"into it."
			Reply(char, txt)
		case oper == "fstat":
			txt = "Syntax: tell katumi fstat <stat> <sym> <num>" +
				"[, <stat2> <sym2> <num2>][, resist <resist>] -- " +
				"Example: tell katumi fstat maxagi > 0, resist fire -- " +
				"Katumi provides up to 10 results which match the parameters."
			Reply(char, txt)
			txt = "Type attribs as they appear in stats: str, maxstr, svsp," +
				" sf_illu, fire, unarm, etc. Valid comparisons are >, <, and =." +
				" Resists check for a positive value. " +
				"Other options will be added later."
			Reply(char, txt)
		default:
			txt = NotFound("help file", oper)
			Reply(char, txt)
		}
	case cmd == "stat" && oper != "":
		FindItem(oper, "short_stats")
	case cmd == "astat" && oper != "":
		FindItem(oper, "long_stats")
	case cmd == "fstat" && oper != "":
		opers := strings.Split(oper, ", ")
		query := "SELECT short_stats FROM items"
		var args []interface{}
		for _, ops := range opers {
			fop := strings.Fields(ops)
			switch {
			case len(fop) == 3:
				fop[0] = strings.ToLower(fop[0])
				comp := fop[1]
				if strings.ContainsAny(comp, "=<>") {
					if !strings.Contains(query, "WHERE") {
						query += " WHERE item_id IN"
					} else {
						query += " AND item_id IN"
					}
					query += " (SELECT i.item_id FROM items i, item_attribs a " +
						"WHERE i.item_id = a.item_id " +
						"AND attrib_abbr = ? AND attrib_value " + comp + " ?)"
					args = append(args, fop[0], fop[2])
				}
			case len(fop) == 2:
				if strings.ToLower(fop[0]) == "resist" {
					res := strings.ToLower(fop[1])
					if !strings.Contains(query, "WHERE") {
						query += " WHERE item_id IN"
					} else {
						query += " AND item_id IN"
					}
					query += " (SELECT i.item_id FROM items i, item_resists r " +
						"WHERE i.item_id = r.item_id " +
						"AND resist_abbr = ? AND resist_value > 0)"
					args = append(args, res)
				}
			}
		}
		query += " LIMIT 10;"
		if strings.Contains(query, "WHERE") {
			db, err := sql.Open("sqlite3", "toril.db")
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			// debugging
			//fmt.Printf("Query : %s\n", query)
			//fmt.Printf("Params: %s\n", args)

			stmt, err := db.Prepare(query)
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()

			rows, err := stmt.Query(args...)
			if err != nil {
				log.Fatal(err)
			}
			defer rows.Close()

			var replied bool
			for rows.Next() {
				var stats string
				err = rows.Scan(&stats)
				Reply(char, stats)
				replied = true
			}
			if !replied {
				Reply(char, NotFound("item(s)", oper))
			}
			err = rows.Err()
			if err != nil {
				log.Fatal(err)
			}
			rows.Close()
		} else {
			Reply(char, syntax)
		}
	case cmd == "who" && oper != "":
		fmt.Println()
	case cmd == "clist" && oper != "":
		fmt.Println()
	case cmd == "char" && oper != "":
		fmt.Println()
	case cmd == "find" && oper != "":
		fmt.Println()
	case cmd == "class" && oper != "":
		fmt.Println()
	case cmd == "delalt" && oper != "":
		fmt.Println()
	case cmd == "addalt" && oper != "":
		fmt.Println()
	case cmd == "lr":
		fmt.Println()
	case cmd == "lrdel" && oper != "":
		fmt.Println()
	default:
		Reply(char, syntax)
	}
}
