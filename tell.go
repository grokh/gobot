package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strconv"
	"strings"
	"time"
)

func NotFound(four string, oper string) string {
	return "404 " + four + " not found: " + oper
}

func Reply(char string, msg string) {
	fmt.Printf("t %s %s\n", char, msg)
}

func FindItem(oper string, length string) string {
	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var stats string

	// query items table for exact item name
	item := oper
	query := "SELECT " + length + " FROM items " +
		"WHERE item_name = ? LIMIT 1"

	stmt, err := db.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	err = stmt.QueryRow(item).Scan(&stats)
	if err == sql.ErrNoRows {
		// if no exact match on item name, check LIKE
		item = "%" + oper + "%"
		query = "SELECT " + length + " FROM items " +
			"WHERE LOWER(item_name) LIKE LOWER(?) LIMIT 1"

		stmt2, err := db.Prepare(query)
		if err != nil {
			log.Fatal(err)
		}
		defer stmt2.Close()

		err = stmt2.QueryRow(item).Scan(&stats)
		if err == sql.ErrNoRows {
			// if no match on ILIKE, check with %'s in place of spaces
			item = " " + oper + " "
			item = strings.Replace(item, " ", "%", -1)
			query = "SELECT " + length + " FROM items " +
				"WHERE LOWER(item_name) LIKE LOWER(?) LIMIT 1"

			stmt3, err := db.Prepare(query)
			if err != nil {
				log.Fatal(err)
			}
			defer stmt3.Close()

			err = stmt3.QueryRow(item).Scan(&stats)
			if err == sql.ErrNoRows {
				// if no match on %'s, check general strings in any order
				words := strings.Fields(oper)
				var args []interface{}
				query = "SELECT " + length + " FROM " +
					"items WHERE "
				for _, word := range words {
					query += "LOWER(item_name) LIKE LOWER(?) AND "
					args = append(args, "%"+word+"%")
				}
				query = strings.TrimRight(query, "AND ")

				stmt4, err := db.Prepare(query)
				if err != nil {
					log.Fatal(err)
				}
				defer stmt4.Close()

				err = stmt4.QueryRow(args...).Scan(&stats)
				/* mattn/go-sqlite3 crashes when accessing FTS3 tables
				query = "SELECT " + length + " FROM " +
					"items WHERE item_id = " +
					"(SELECT item_id FROM item_search " +
					"WHERE item_text MATCH ?) " +
					"LIMIT 1"

				stmt4, err := db.Prepare(query)
				if err != nil {
					log.Fatal(err)
				}
				defer stmt4.Close()

				err = stmt4.QueryRow(oper).Scan(&stats)
				*/
				if err == sql.ErrNoRows {
					stats = NotFound("item", oper)
				} else if err != nil {
					log.Fatal(err)
				}
			} else if err != nil {
				log.Fatal(err)
			}
		} else if err != nil {
			log.Fatal(err)
		}
	} else if err != nil {
		log.Fatal(err)
	}
	return stats
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
	//fmt.Printf("Cmd : %s\n", cmd)
	//fmt.Printf("Oper: %s\n", oper)

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
		txt = FindItem(oper, "short_stats")
		Reply(char, txt)
	case cmd == "astat" && oper != "":
		txt = FindItem(oper, "long_stats")
		Reply(char, txt)
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
				txt = NotFound("item(s)", oper)
				Reply(char, txt)
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
		db, err := sql.Open("sqlite3", "toril.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		query := "SELECT account_name, char_name " +
			"FROM chars WHERE vis = 't' " +
			"AND (account_name = " +
			"(SELECT account_name FROM chars " +
			"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
			"OR LOWER(account_name) = LOWER(?)) " +
			"ORDER BY char_level DESC, char_name ASC"

		rows, err := db.Query(query, oper, oper)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		for rows.Next() {
			rows.Scan(&Char.acct, &Char.name)
			if strings.Contains(txt, "@") {
				txt += ", " + Char.name
			} else {
				txt = "@" + Char.acct
			}
		}
		rows.Close()

		if strings.Contains(txt, "@") {
			Reply(char, txt)
		} else {
			txt = NotFound("character or account", oper)
			Reply(char, txt)
		}
	case cmd == "clist" && oper != "":
		db, err := sql.Open("sqlite3", "toril.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		query := "SELECT char_level, class_name, char_name, char_race, " +
			"account_name, DATETIME(last_seen) " +
			"FROM chars WHERE vis = 't' " +
			"AND (account_name = " +
			"(SELECT account_name FROM chars " +
			"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
			"OR LOWER(account_name) = LOWER(?)) " +
			"ORDER BY char_level DESC, char_name ASC"

		rows, err := db.Query(query, oper, oper)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var replied bool
		for rows.Next() {
			var seen string
			rows.Scan(&Char.lvl, &Char.class, &Char.name, &Char.race, &Char.acct, &seen)
			txt = fmt.Sprintf(
				"[%d %s] %s (%s) (@%s) seen %s",
				Char.lvl, Char.class, Char.name, Char.race, Char.acct, seen,
			)
			Reply(char, txt)
			replied = true
		}
		rows.Close()
		if !replied {
			txt = NotFound("character or account", oper)
			Reply(char, txt)
		}
	case cmd == "char" && oper != "":
		db, err := sql.Open("sqlite3", "toril.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		query := "SELECT char_level, class_name, char_name, char_race, " +
			"account_name, DATETIME(last_seen) " +
			"FROM chars WHERE vis = 't' " +
			"AND LOWER(char_name) = LOWER(?)"
		stmt, err := db.Prepare(query)
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		var seen string
		err = stmt.QueryRow(oper).Scan(&Char.lvl, &Char.class, &Char.name, &Char.race, &Char.acct, &seen)
		if err == sql.ErrNoRows {
			txt = NotFound("character", oper)
			Reply(char, txt)
		} else if err != nil {
			log.Fatal(err)
		} else {
			txt = fmt.Sprintf(
				"[%d %s] %s (%s) (@%s) seen %s",
				Char.lvl, Char.class, Char.name, Char.race, Char.acct, seen,
			)
			Reply(char, txt)
		}
	case cmd == "find" && oper != "":
		db, err := sql.Open("sqlite3", "toril.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		query := "SELECT account_name, char_name, " +
			"(STRFTIME('%s','now','localtime') - STRFTIME('%s',last_seen)) seconds " +
			"FROM chars WHERE vis = 't' " +
			"AND (account_name = " +
			"(SELECT account_name FROM chars " +
			"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
			"OR LOWER(account_name) = LOWER(?)) " +
			"ORDER BY last_seen DESC LIMIT 1"
		stmt, err := db.Prepare(query)
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		var seconds int
		err = stmt.QueryRow(oper, oper).Scan(&Char.acct, &Char.name, &seconds)
		if err == sql.ErrNoRows {
			txt = NotFound("character or account", oper)
			Reply(char, txt)
		} else if err != nil {
			log.Fatal(err)
		} else {
			var seen string
			online := false
			secs := time.Duration(seconds) * time.Second
			if secs.Hours() >= 24 && secs.Hours() < 48 {
				seen = "1 day"
			} else if secs.Hours() >= 48 {
				days := int(secs.Hours()) / 24
				seen = fmt.Sprintf("%d days", days)
			} else if secs.Seconds() > 3600 {
				hours := int(secs.Seconds()) / 3600
				minutes := int(secs.Seconds()) % 3600
				seen = fmt.Sprintf("%dh%dm", hours, minutes)
			} else if secs.Seconds() > 60 {
				minutes := int(secs.Seconds()) / 60
				seconds = int(secs.Seconds()) % 60
				seen = fmt.Sprintf("%dm%ds", minutes, seconds)
			} else if secs.Seconds() <= 60 {
				seen = fmt.Sprintf("%ds", int(secs.Seconds()))
				online = true
			} else {
				log.Fatalf("Error: seconds were %d", secs.Seconds())
			}
			// Char.seen = secs.String() // easier :/
			if !online {
				txt = fmt.Sprintf("@%s last seen %s ago as %s", Char.acct, seen, Char.name)
			} else {
				txt = fmt.Sprintf("@%s is online as %s", Char.acct, Char.name)
			}
			Reply(char, txt)
		}
	case cmd == "class" && oper != "":
		db, err := sql.Open("sqlite3", "toril.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		loc, err := time.LoadLocation("America/New_York")
		if err != nil {
			log.Fatal(err)
		}
		date := time.Now().In(loc).Add(-time.Minute)
		query := "SELECT char_name, class_name, char_race, char_level, account_name " +
			"FROM chars WHERE LOWER(class_name) = LOWER(?) AND vis = 't' " +
			"AND account_name IN " +
			"(SELECT account_name FROM chars " +
			"WHERE last_seen > ? " +
			"AND vis = 't') " +
			"ORDER BY char_level DESC"

		rows, err := db.Query(query, oper, date)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var replied bool
		for rows.Next() {
			rows.Scan(&Char.name, &Char.class, &Char.race, &Char.lvl, &Char.acct)
			txt = fmt.Sprintf(
				"[%d %s] %s (%s) (@%s)",
				Char.lvl, Char.class, Char.name, Char.race, Char.acct,
			)
			Reply(char, txt)
			replied = true
		}
		rows.Close()
		if !replied {
			txt = NotFound("class", oper)
			Reply(char, txt)
		}
	case cmd == "delalt" && oper != "":
		db, err := sql.Open("sqlite3", "toril.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		query := "SELECT account_name, char_name FROM chars " +
			"WHERE LOWER(char_name) = LOWER(?) " +
			"AND vis = 't' AND account_name = " +
			"(SELECT account_name FROM chars WHERE char_name = ?)"

		stmt, err := db.Prepare(query)
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		err = stmt.QueryRow(oper, char).Scan(&Char.acct, &Char.name)
		if err == sql.ErrNoRows {
			txt = NotFound("character or account", oper)
			Reply(char, txt)
		} else if err != nil {
			log.Fatal(err)
		} else {
			query = "UPDATE chars SET vis = 'f' " +
				"WHERE LOWER(char_name) = LOWER(?)"

			tx, err := db.Begin()
			if err != nil {
				log.Fatal(err)
			}
			stmt, err := tx.Prepare(query)
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()

			_, err = stmt.Exec(oper)
			if err != nil {
				log.Fatal(err)
			}
			tx.Commit()
			txt = fmt.Sprintf("Removed character from your alt list:: %s", oper)
			Reply(char, txt)
		}
	case cmd == "addalt" && oper != "":
		db, err := sql.Open("sqlite3", "toril.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		query := "SELECT account_name, char_name FROM chars " +
			"WHERE LOWER(char_name) = LOWER(?) " +
			"AND account_name = " +
			"(SELECT account_name FROM chars WHERE char_name = ?)"

		stmt, err := db.Prepare(query)
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		err = stmt.QueryRow(oper, char).Scan(&Char.acct, &Char.name)
		if err == sql.ErrNoRows {
			txt = NotFound("character or account", oper)
			Reply(char, txt)
		} else if err != nil {
			log.Fatal(err)
		} else {
			query = "UPDATE chars SET vis = 't' " +
				"WHERE LOWER(char_name) = LOWER(?)"

			tx, err := db.Begin()
			if err != nil {
				log.Fatal(err)
			}
			stmt, err := tx.Prepare(query)
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()

			_, err = stmt.Exec(oper)
			if err != nil {
				log.Fatal(err)
			}
			tx.Commit()
			txt = fmt.Sprintf("Re-added character to your alt list:: %s", oper)
			Reply(char, txt)
		}
	case cmd == "lr":
		db, err := sql.Open("sqlite3", "toril.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		if oper == "" {
			query := "SELECT char_name, report_text, " +
				"DATETIME(report_time) " +
				"FROM loads WHERE boot_id = " +
				"(SELECT MAX(boot_id) FROM boots)" +
				"AND deleted = 'f'"

			rows, err := db.Query(query)
			if err != nil {
				log.Fatal(err)
			}
			defer rows.Close()

			var replied bool
			counter := 1
			for rows.Next() {
				var report string
				var date string
				err = rows.Scan(&Char.name, &report, &date)
				txt = fmt.Sprintf(
					"%d: %s [%s at %s]",
					counter, report, Char.name, date,
				)
				Reply(char, txt)
				counter++
				replied = true
			}
			err = rows.Err()
			if err != nil {
				log.Fatal(err)
			}
			if !replied {
				txt = "No loads reported for current boot."
				Reply(char, txt)
			}
			rows.Close()
		} else if strings.Contains(oper, " ") {
			tx, err := db.Begin()
			if err != nil {
				log.Fatal(err)
			}
			loc, err := time.LoadLocation("America/New_York")
			if err != nil {
				log.Fatal(err)
			}

			date := time.Now().In(loc)
			query := "INSERT INTO loads " +
				"VALUES((SELECT MAX(boot_id) FROM boots), ?, ?, ?, 'f')"

			stmt, err := tx.Prepare(query)
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()

			_, err = stmt.Exec(date, oper, char)
			if err != nil {
				log.Fatal(err)
			}
			tx.Commit()
			txt = fmt.Sprintf("Load reported: %s", oper)
			Reply(char, txt)
		} else {
			Reply(char, syntax)
		}
	case cmd == "lrdel" && oper != "":
		db, err := sql.Open("sqlite3", "toril.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		query := "SELECT boot_id, report_time, report_text FROM loads " +
			"WHERE deleted = 'f' " +
			"AND boot_id = (SELECT MAX(boot_id) FROM boots)"

		rows, err := db.Query(query)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		num, err := strconv.Atoi(oper)
		if err != nil {
			Reply(char, syntax)
			return
		}
		var curboot string
		var rtime time.Time
		var report string
		counter := 1
		for rows.Next() {
			var boot string
			var date time.Time
			var text string
			err = rows.Scan(&boot, &date, &text)
			if counter == num {
				curboot = boot
				rtime = date
				report = text
			}
			counter++
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}
		if counter > 1 {
			query = "UPDATE loads SET deleted = 't' " +
				"WHERE boot_id = ? AND report_time = ?"

			tx, err := db.Begin()
			if err != nil {
				log.Fatal(err)
			}
			stmt, err := tx.Prepare(query)
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()

			_, err = stmt.Exec(curboot, rtime)
			if err != nil {
				log.Fatal(err)
			}
			tx.Commit()
			txt = fmt.Sprintf("Load deleted: %s", report)
			Reply(char, txt)
		} else {
			txt = "No loads reported for current boot."
			Reply(char, txt)
		}
	default:
		Reply(char, syntax)
	}
}
