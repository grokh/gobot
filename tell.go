package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

func NotFound(four string, oper string) string {
	log.Printf("404: %s: %s\n", four, oper)
	return "404 " + four + " not found: " + oper
}

func FindItem(oper string, length string) []string {
	db := OpenDB()
	defer db.Close()

	var res []string
	count := 0
	// query items table for exact item name
	txt := oper
	query := "SELECT " + length + " FROM items " +
		"WHERE item_name = ? LIMIT 12"

	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err := stmt.Query(txt)
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		res = append(res, "")
		err = rows.Scan(&res[count])
		count++
	}
	ChkRows(rows)
	stmt.Close()

	if count == 0 {
		//fmt.Println("No exact match")
		// if no exact match on item name, check LIKE
		txt = "%" + oper + "%"
		query = "SELECT item_name, " + length + " FROM items " +
			"WHERE item_name LIKE ? LIMIT 1"

		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		rows, err = stmt.Query(txt)
		ChkErr(err)
		defer rows.Close()

		for rows.Next() {
			res = append(res, "")
			err = rows.Scan(&txt, &res[count])
			count++
		}
		ChkRows(rows)
		stmt.Close()
	}

	if count == 0 {
		//fmt.Println("No like match")
		// if no match on LIKE, check with %'s in place of spaces
		txt = " " + oper + " "
		txt = strings.Replace(txt, " ", "%", -1)
		query = "SELECT item_name, " + length + " FROM items " +
			"WHERE item_name LIKE ? LIMIT 1"

		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		rows, err = stmt.Query(txt)
		ChkErr(err)
		defer rows.Close()

		for rows.Next() {
			res = append(res, "")
			err = rows.Scan(&txt, &res[count])
			count++
		}
		ChkRows(rows)
		stmt.Close()
	}

	if count == 0 {
		//fmt.Println("No in order match")
		// if no match on %'s, check general strings in any order
		words := strings.Fields(oper)
		args := make([]interface{}, len(words))
		query = "SELECT item_name, " + length + " FROM " +
			"items WHERE "
		for n, word := range words {
			query += "item_name LIKE ? AND "
			args[n] = "%" + word + "%"
		}
		query = strings.TrimRight(query, "AND ")
		query += " LIMIT 1"

		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		rows, err = stmt.Query(args...)
		ChkErr(err)
		defer rows.Close()

		for rows.Next() {
			res = append(res, "")
			err = rows.Scan(&txt, &res[count])
			count++
		}
		ChkRows(rows)
		stmt.Close()
	}

	if count == 0 {
		//fmt.Println("No out of order match")
		res = append(res, NotFound("item", oper))
	}

	return res
}

func Help(oper string) []string {
	oper = strings.ToLower(oper)
	var txt []string
	switch oper {
	case "?":
		txt = append(txt, "Syntax: tell katumi ? -- "+
			"Katumi provides a full listing of valid commands.")
	case "hidden?":
		txt = append(txt, "Syntax: tell katumi hidden? -- "+
			"Katumi sends a tell in reply "+
			"if she can see you. If you receive no reply, you are hidden. "+
			"Katumi has permanent detect invis to ensure that won't cause "+
			"issues.")
	case "who":
		txt = append(txt, "Syntax: tell katumi who <acct/char> -- "+
			"Example: tell katumi who rynshana -- "+
			"Katumi provides the account name along with a list of every "+
			"known alt of the named character as a reply. Also works with "+
			"account names.")
	case "char":
		txt = append(txt, "Syntax: tell katumi char <char> -- "+
			"Example: tell katumi char rynshana -- "+
			"Katumi provides the account name along with full information "+
			"on the named character as a reply, "+
			"to include level, class, race, and date/time last seen.")
	case "find":
		txt = append(txt, "Syntax: tell katumi find <acct/char> -- "+
			"Example: tell katumi find rynshana -- "+
			"Katumi provides the account name along with the last known "+
			"sighting of any of that character's alts. "+
			"If they have an alt online, "+
			"the time will measure in seconds. "+
			"Also works with account names.")
	case "clist":
		txt = append(txt, "Syntax: tell katumi clist <acct/char> -- "+
			"Example: tell katumi clist rynshana -- "+
			"Katumi provides a full "+
			"listing of every known alt belonging to <char>, including race, "+
			"class, level, and date/time last seen, matching the format of "+
			"the 'char' command. Also works with account names.")
	case "class":
		txt = append(txt, "Syntax: tell katumi class <class> -- "+
			"Example: tell katumi class enchanter -- "+
			"Katumi provides a "+
			"list of alts matching the named class for characters who "+
			"are currently online, letting group leaders find useful "+
			"alts from the 'who' list.")
	case "delalt":
		txt = append(txt, "Syntax: tell katumi delalt <char> -- "+
			"Example: tell katumi delalt rynshana -- "+
			"Katumi no longer "+
			"provides information on the alt, removing it from 'clist', "+
			"'who', and 'find' commands. Only works for characters "+
			"attached to the same account requesting the removal.")
	case "addalt":
		txt = append(txt, "Syntax: tell katumi addalt <char> -- "+
			"Example: tell katumi addalt rynshana -- "+
			"Katumi begins "+
			"providing information on the named alt, who had previously "+
			"been removed with 'delalt', adding the character back to "+
			"'clist', 'who', and 'find' commands. Only works for chars "+
			"attached to the same account.")
	case "lr":
		txt = append(txt, "Syntax: tell katumi lr -- "+
			"Katumi provides a list of load "+
			"reports for the current boot. This could be rares or quests "+
			"other players have found or completed. Use the 'lrdel' command "+
			"to remove bad or out of date reports.")
		txt = append(txt, "Syntax: tell katumi lr <report> -- "+
			"Example: tell katumi lr timestop gnome at ako village -- "+
			"Katumi adds <report> "+
			"to the list of load reports for the current boot. If you find "+
			"a rare, global load, or complete a quest or the like, report "+
			"it along with a location so other players will know!")
	case "lrdel":
		txt = append(txt, "Syntax: tell katumi lrdel <num> -- "+
			"Example: tell katumi lrdel 3 -- "+
			"Katumi removes the "+
			"numbered item from the load reports, if a quest is completed "+
			"or a rare killed, or a report found to be inaccurate. Please "+
			"do not abuse this command - this service helps everyone.")
	case "stat":
		txt = append(txt, "Syntax: tell katumi stat <item> -- "+
			"Example: tell katumi stat isha cloak -- "+
			"Katumi provides stat info for the item named. "+
			"Use 'astat' for full text of acronyms and keywords. "+
			"The name search is fairly forgiving. Please send new stats "+
			"in an mwrite to katumi or email to kristi.michaels@gmail.com")
	case "astat":
		txt = append(txt, "Syntax: tell katumi astat <item> -- "+
			"Example tell katumi astat destruction sword -- "+
			"Katumi provides full "+
			"stat information for the item named. Use 'stat' for short "+
			"text. The name search is fairly forgiving, though the stats "+
			"are a little buggy right now since I haven't put much time "+
			"into it.")
	case "fstat":
		txt = append(txt, "Syntax: tell katumi fstat <fields> -- "+
			"Attempts to replicate the Advanced Search from TorilEQ website. "+
			"Katumi provides up to 10 results which match the parameters."+
			"Valid fields are attributes, resists, and slots. "+
			"Create a multi-field query using a comma and a space.")
		txt = append(txt, "Valid symbols: >, <, or = -- Valid stats:"+
			"-- Syntax: <stat> <sym> <num>")
		txt = append(txt, "Valid resists: "+
			"-- Syntax: resist <resist>")
		txt = append(txt, "Valid slots: "+
			"-- Syntax: slot <slot>")
		txt = append(txt, "Example using one attribute, one resist, "+
			"and one slot in a single combined query -- "+
			"tell katumi fstat maxagi > 0, resist fire, slot ear")
		txt = append(txt, "Syntax: tell katumi fstat <stat> <sym> <num>"+
			"[, <stat2> <sym2> <num2>][, resist <resist>][, slot <slot>] -- "+
			"Example: tell katumi fstat maxagi > 0, resist fire, slot ear -- "+
			"Katumi provides up to 10 results which match the parameters.")
		txt = append(txt,
			"Type attribs as they appear in stats: str, maxstr, svsp, "+
				"dam, sf_illu, fire, unarm, ear, on_body, etc. "+
				"Valid comparisons are >, <, and =. "+
				"Resists check for a positive value. "+
				"Other options will be added later.")
	default:
		txt = append(txt, NotFound("help file", oper))
	}
	return txt
}

func Fstat(oper string) []string {
	var txts []string
	opers := strings.Split(oper, ", ")
	query := "SELECT short_stats FROM items"
	var args []interface{}
	for _, ops := range opers {
		fop := strings.Fields(ops)
		switch len(fop) {
		case 3:
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
		case 2:
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
			} else if strings.ToLower(fop[0]) == "slot" {
				slot := "%" + strings.ToLower(fop[1]) + "%"
				if !strings.Contains(query, "WHERE") {
					query += " WHERE item_id IN"
				} else {
					query += " AND item_id IN"
				}
				query += " (SELECT i.item_id FROM items i, item_slots s " +
					"WHERE i.item_id = s.item_id " +
					"AND slot_abbr LIKE ?)"
				args = append(args, slot)
			}
		}
	}
	query += " LIMIT 10;"
	if strings.Contains(query, "WHERE") {
		db := OpenDB()
		defer db.Close()

		// debugging
		//log.Printf("Query : %s\n", query)
		//log.Printf("Params: %s\n", args)

		stmt, err := db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		rows, err := stmt.Query(args...)
		ChkErr(err)
		defer rows.Close()

		var found bool
		for rows.Next() {
			var stats string
			err = rows.Scan(&stats)
			txts = append(txts, stats)
			found = true
		}
		if !found {
			txts = append(txts, NotFound("item(s)", oper))
		}
		ChkRows(rows)
	} else {
		txts = append(txts, BadSyntax)
	}
	return txts
}

func Who(oper string) string {
	db := OpenDB()
	defer db.Close()

	var txt string
	query := "SELECT account_name, char_name " +
		"FROM chars WHERE vis = 't' " +
		"AND (account_name = " +
		"(SELECT account_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
		"OR LOWER(account_name) = LOWER(?)) " +
		"ORDER BY char_level DESC, char_name ASC"

	rows, err := db.Query(query, oper, oper)
	ChkErr(err)
	defer rows.Close()

	var c Char
	for rows.Next() {
		err = rows.Scan(&c.acct, &c.name)
		if strings.Contains(txt, "@") {
			txt += ", " + c.name
		} else {
			txt = "@" + c.acct
			txt += ": " + c.name
		}
	}
	ChkRows(rows)

	if !strings.Contains(txt, "@") {
		txt = NotFound("character or account", oper)
	}
	return txt
}

func Clist(oper string) []string {
	db := OpenDB()
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
	ChkErr(err)
	defer rows.Close()

	var txts []string
	var found bool
	var c Char
	for rows.Next() {
		err = rows.Scan(&c.lvl, &c.class, &c.name,
			&c.race, &c.acct, &c.seen)
		char := fmt.Sprintf(
			"[%d %s] %s (%s) (@%s) seen %s",
			c.lvl, c.class, c.name, c.race, c.acct, c.seen,
		)
		txts = append(txts, char)
		found = true
	}
	ChkRows(rows)
	if !found {
		txts = append(txts, NotFound("character or account", oper))
	}
	return txts
}

func CharInfo(oper string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT char_level, class_name, char_name, char_race, " +
		"account_name, DATETIME(last_seen) " +
		"FROM chars WHERE vis = 't' " +
		"AND LOWER(char_name) = LOWER(?)"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var c Char
	err = stmt.QueryRow(oper).Scan(
		&c.lvl, &c.class, &c.name, &c.race, &c.acct, &c.seen)
	if err == sql.ErrNoRows {
		txt = NotFound("character", oper)
	} else if err != nil {
		log.Fatal(err)
	} else {
		txt = fmt.Sprintf(
			"[%d %s] %s (%s) (@%s) seen %s",
			c.lvl, c.class, c.name, c.race, c.acct, c.seen,
		)
	}
	return txt
}

func Find(oper string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT account_name, char_name, " +
		"(STRFTIME('%s','now') - STRFTIME('%s',last_seen)) seconds " +
		"FROM chars WHERE vis = 't' " +
		"AND (account_name = " +
		"(SELECT account_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
		"OR LOWER(account_name) = LOWER(?)) " +
		"ORDER BY last_seen DESC LIMIT 1"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var seconds int
	var c Char
	err = stmt.QueryRow(oper, oper).Scan(&c.acct, &c.name, &seconds)
	if err == sql.ErrNoRows {
		txt = NotFound("character or account", oper)
	} else if err != nil {
		log.Fatal(err)
	} else {
		online := false
		secs := time.Duration(seconds) * time.Second
		if secs.Hours() >= 24 && secs.Hours() < 48 {
			c.seen = "1 day"
		} else if secs.Hours() >= 48 {
			days := int(secs.Hours()) / 24
			c.seen = fmt.Sprintf("%d days", days)
		} else if secs.Seconds() > 3600 {
			hours := int(secs.Seconds()) / 3600
			minutes := (int(secs.Seconds()) % 3600) / 60
			c.seen = fmt.Sprintf("%dh%dm", hours, minutes)
		} else if secs.Seconds() > 60 {
			minutes := int(secs.Seconds()) / 60
			seconds = int(secs.Seconds()) % 60
			c.seen = fmt.Sprintf("%dm%ds", minutes, seconds)
		} else if secs.Seconds() <= 60 && secs.Seconds() >= 0 {
			c.seen = fmt.Sprintf("%ds", int(secs.Seconds()))
			online = true
		} else {
			log.Printf("'find' error: seconds were %d\n", secs.Seconds())
		}
		// Char.seen = secs.String() // easier :/
		if !online {
			txt = fmt.Sprintf(
				"@%s last seen %s ago as %s",
				c.acct, c.seen, c.name)
		} else {
			txt = fmt.Sprintf(
				"@%s is online as %s",
				c.acct, c.name)
		}
	}
	return txt
}

func Name(oper string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT account_name, player_name " +
		"FROM accounts " +
		"WHERE player_name IS NOT NULL " +
		"AND (account_name = " +
		"(SELECT account_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
		"OR LOWER(account_name) = LOWER(?))"

	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var c Char
	err = stmt.QueryRow(oper, oper).Scan(&c.acct, &c.name)
	if err == sql.ErrNoRows {
		query := "SELECT account_name " +
			"FROM accounts " +
			"WHERE (account_name = " +
			"(SELECT account_name FROM chars " +
			"WHERE LOWER(char_name) = LOWER(?) AND vis = 't') " +
			"OR LOWER(account_name) = LOWER(?))"

		stmt, err := db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		err = stmt.QueryRow(oper, oper).Scan(&c.acct)
		if err == sql.ErrNoRows {
			txt = NotFound("character or account", oper)
		} else if err != nil {
			log.Fatal(err)
		} else {
			txt = fmt.Sprintf("@%s did not disclose their real name", c.acct)
		}
	} else if err != nil {
		log.Fatal(err)
	} else {
		txt = fmt.Sprintf("@%s's real name is %s", c.acct, c.name)
	}
	return txt
}

func FindClass(oper string) []string {
	db := OpenDB()
	defer db.Close()

	loc, err := time.LoadLocation("America/New_York")
	ChkErr(err)
	date := time.Now().In(loc).Add(-time.Minute)
	query := "SELECT char_name, class_name, char_race, " +
		"char_level, account_name " +
		"FROM chars WHERE LOWER(class_name) = LOWER(?) " +
		"AND vis = 't' " +
		"AND account_name IN " +
		"(SELECT account_name FROM chars " +
		"WHERE last_seen > ? " +
		"AND vis = 't') " +
		"ORDER BY char_level DESC"

	rows, err := db.Query(query, oper, date)
	ChkErr(err)
	defer rows.Close()

	var txts []string
	var found bool
	var c Char
	for rows.Next() {
		err = rows.Scan(&c.name, &c.class, &c.race,
			&c.lvl, &c.acct)
		txt := fmt.Sprintf(
			"[%d %s] %s (%s) (@%s)",
			c.lvl, c.class, c.name, c.race, c.acct,
		)
		txts = append(txts, txt)
		found = true
	}
	ChkRows(rows)
	if !found {
		txts = append(txts, NotFound("class", oper))
	}
	return txts
}

func DelAlt(oper string, char string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT account_name, char_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) " +
		"AND vis = 't' AND account_name = " +
		"(SELECT account_name FROM chars WHERE char_name = ?)"

	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var c Char
	err = stmt.QueryRow(oper, char).Scan(&c.acct, &c.name)
	if err == sql.ErrNoRows {
		txt = NotFound("character or account", oper)
	} else if err != nil {
		log.Fatal(err)
	} else {
		query = "UPDATE chars SET vis = 'f' " +
			"WHERE LOWER(char_name) = LOWER(?)"

		tx, err := db.Begin()
		ChkErr(err)
		stmt, err := tx.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		_, err = stmt.Exec(oper)
		ChkErr(err)
		tx.Commit()
		log.Printf("Delalt: %s\n", oper)
		txt = fmt.Sprintf("Removed character from your alt list:: %s", oper)
	}
	return txt
}

func AddAlt(oper string, char string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT account_name, char_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?) " +
		"AND account_name = " +
		"(SELECT account_name FROM chars WHERE char_name = ?)"

	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var c Char
	err = stmt.QueryRow(oper, char).Scan(&c.acct, &c.name)
	if err == sql.ErrNoRows {
		txt = NotFound("character or account", oper)
	} else if err != nil {
		log.Fatal(err)
	} else {
		query = "UPDATE chars SET vis = 't' " +
			"WHERE LOWER(char_name) = LOWER(?)"

		tx, err := db.Begin()
		ChkErr(err)
		stmt, err := tx.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		_, err = stmt.Exec(oper)
		ChkErr(err)
		tx.Commit()
		txt = fmt.Sprintf("Re-added character to your alt list:: %s", oper)
	}
	return txt
}

func AddName(oper string, char string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT account_name " +
		"FROM chars " +
		"WHERE char_name = ?"

	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var txt string
	var c Char
	err = stmt.QueryRow(char).Scan(&c.acct)
	if err == sql.ErrNoRows {
		txt = NotFound("character", char)
	} else if err != nil {
		log.Fatal(err)
	} else {
		query = "UPDATE accounts " +
			"SET player_name = ? " +
			"WHERE account_name = ?"

		tx, err := db.Begin()
		ChkErr(err)
		stmt, err := tx.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		name := strings.Title(oper)
		_, err = stmt.Exec(name, c.acct)
		ChkErr(err)
		tx.Commit()
		txt = fmt.Sprintf("Your real name recorded as: %s", oper)
	}
	return txt
}

func LoadReport(oper string, char string) []string {
	db := OpenDB()
	defer db.Close()

	var txts []string
	var txt string
	var c Char
	if oper == "" {
		query := "SELECT char_name, report_text, " +
			"DATETIME(report_time) " +
			"FROM loads WHERE boot_id = " +
			"(SELECT MAX(boot_id) FROM boots)" +
			"AND deleted = 'f'"

		rows, err := db.Query(query)
		ChkErr(err)
		defer rows.Close()

		var found bool
		counter := 1
		for rows.Next() {
			var report string
			var date string
			err = rows.Scan(&c.name, &report, &date)
			txt = fmt.Sprintf(
				"%d: %s [%s at %s]",
				counter, report, c.name, date,
			)
			txts = append(txts, txt)
			counter++
			found = true
		}
		ChkRows(rows)
		if !found {
			txts = append(txts, "No loads reported for current boot.")
		}
	} else if strings.Contains(oper, " ") {
		tx, err := db.Begin()
		ChkErr(err)
		loc, err := time.LoadLocation("America/New_York")
		ChkErr(err)

		date := time.Now().In(loc)
		query := "INSERT INTO loads " +
			"VALUES((SELECT MAX(boot_id) FROM boots), ?, ?, ?, 'f')"

		stmt, err := tx.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		_, err = stmt.Exec(date, oper, char)
		ChkErr(err)
		tx.Commit()
		txts = append(txts, fmt.Sprintf("Load reported: %s", oper))
	} else {
		txts = append(txts, BadSyntax)
	}
	return txts
}

func LRDel(oper string) string {
	db := OpenDB()
	defer db.Close()

	query := "SELECT boot_id, report_time, report_text FROM loads " +
		"WHERE deleted = 'f' " +
		"AND boot_id = (SELECT MAX(boot_id) FROM boots)"

	rows, err := db.Query(query)
	ChkErr(err)
	defer rows.Close()

	var txt string
	num, err := strconv.Atoi(oper)
	if err != nil {
		return BadSyntax
	}
	var curboot string
	var rtime time.Time
	report := ""
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
	ChkRows(rows)
	if counter > 1 {
		if report != "" {
			query = "UPDATE loads SET deleted = 't' " +
				"WHERE boot_id = ? AND report_time = ?"

			tx, err := db.Begin()
			ChkErr(err)
			stmt, err := tx.Prepare(query)
			ChkErr(err)
			defer stmt.Close()

			_, err = stmt.Exec(curboot, rtime)
			ChkErr(err)
			tx.Commit()
			txt = fmt.Sprintf("Load deleted: %s", report)
		} else {
			txt = "Invalid load report number."
		}
	} else {
		txt = "No loads reported for current boot."
	}
	return txt
}

var BadSyntax string = "Invalid syntax. For valid syntax: tell katumi ?, " +
	"tell katumi help <cmd>"

func ReplyTo(char string, tell string) []string {
	info := []string{"I am a Helper Bot (Beta). Each command I accept " +
		"has further help files available at: tell katumi help <cmd>",
		"Find items: Acronymed stats: stat <item name>, " +
			"Stats fully spelled out: astat <item name>, " +
			"Find items by attributes, slots, etc.: (for proper usage, " +
			"tell katumi help fstat)",
		"Find people: Provide acct and char info: who <char/acct>, " +
			"clist <char/acct>, char <char>, Show last online alt: " +
			"find <char/acct>, Find alts of listed class for people online: " +
			"class <class>, RL names: name <char/acct>, addname <name>, " +
			"Control your listing: delalt <char>, addalt <char>",
		"Misc: This message: ?, More info on each command: help <cmd>, " +
			"Find out if you're hidden: hidden, Load reports for rares or " +
			"global mobs: lr, lr <report>, lrdel <num>",
	}
	var txt []string

	oper := ""
	split := strings.SplitN(tell, " ", 2)
	cmd := strings.ToLower(split[0])
	if len(split) > 1 {
		oper = strings.TrimSpace(split[1])
	}

	// debugging
	//log.Printf("Cmd : %s\n", cmd)
	//log.Printf("Oper: %s\n", oper)

	switch {
	case cmd == "?":
		txt = info
	case cmd == "help" && oper == "":
		txt = info
	case cmd == "help" && oper != "":
		txt = Help(oper)
	case strings.HasPrefix(cmd, "hidden"):
		if char != "Someone" {
			txt = append(txt, char+" is NOT hidden!")
		}
	case cmd == "stat" && oper != "":
		txt = FindItem(oper, "short_stats")
	case cmd == "astat" && oper != "":
		txt = FindItem(oper, "long_stats")
	case cmd == "fstat" && oper != "":
		txt = Fstat(oper)
	case cmd == "who" && oper != "":
		txt = append(txt, Who(oper))
	case cmd == "clist" && oper != "":
		txt = Clist(oper)
	case cmd == "char" && oper != "":
		txt = append(txt, CharInfo(oper))
	case cmd == "find" && oper != "":
		txt = append(txt, Find(oper))
	case cmd == "name" && oper != "":
		txt = append(txt, Name(oper))
	case cmd == "class" && oper != "":
		txt = FindClass(oper)
	case cmd == "delalt" && oper != "":
		txt = append(txt, DelAlt(oper, char))
	case cmd == "addalt" && oper != "":
		txt = append(txt, AddAlt(oper, char))
	case cmd == "addname" && oper != "":
		txt = append(txt, AddName(oper, char))
	case cmd == "lr":
		txt = LoadReport(oper, char)
	case cmd == "lrdel" && oper != "":
		txt = append(txt, LRDel(oper))
	case cmd == "weather" && oper != "":
		txt = Weather(oper)
	default:
		txt = append(txt, BadSyntax)
	}
	for i, t := range txt {
		if len(t) > 300 {
			if strings.LastIndex(t[:300], " ") != 299 {
				y := strings.Fields(t[:300])
				x, y := y[len(y)-1], y[:len(y)-1]
				a := strings.Join(y, " ")
				b := x + t[300:]
				txt[i] = fmt.Sprintf("t %s %s\n", char, b)
				txt = append(txt, "")
				copy(txt[i+1:], txt[i:])
				txt[i] = fmt.Sprintf("t %s %s\n", char, a)
			} else {
				a := strings.TrimSpace(t[:300])
				b := t[300:]
				txt[i] = fmt.Sprintf("t %s %s\n", char, b)
				txt = append(txt, "")
				copy(txt[i+1:], txt[i:])
				txt[i] = fmt.Sprintf("t %s %s\n", char, a)
			}
		} else {
			txt[i] = fmt.Sprintf("t %s %s\n", char, t)
		}
	}
	return txt
}
