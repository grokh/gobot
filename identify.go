package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Identify(filename string) []string {
	f, err := os.OpenFile(
		"import.sql",
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0640,
	)
	defer f.Close()
	ChkErr(err)
	log.SetOutput(f)

	inserted, ignored := 0, 0

	content, err := ioutil.ReadFile(filename)
	ChkErr(err)
	text := strings.TrimSpace(string(content))

	// put all flags/restricts, or effects, on one line
	re, err := regexp.Compile(`([[:upper:]]{2})\n([[:upper:]]{2})`)
	ChkErr(err)
	text = re.ReplaceAllString(text, "$1 $2")

	// put enchant info on one line
	re, err = regexp.Compile(`\n(Duration)`)
	ChkErr(err)
	text = re.ReplaceAllString(text, " $1")

	// put all resists on same line:
	re, err = regexp.Compile(
		`\n(    [[:alpha:] ]{6}:[[:blank:]]{3,4}[[:digit:]]{1,2}%[ ]?)`)
	ChkErr(err)
	text = re.ReplaceAllString(text, "$1")

	// put first wand/staff/scroll/potion spell on one line:
	re, err = regexp.Compile(`(spell[s]? of:)\n`)
	ChkErr(err)
	text = re.ReplaceAllString(text, "$1 Spell: ")

	// put remaining scroll/potion spells on same line:
	re, err = regexp.Compile(`\n([[:lower:]])`)
	ChkErr(err)
	text = re.ReplaceAllString(text, " Spell: $1")

	items := strings.Split(text, "\n\n")

	// initialize regex checks
	var m []string
	chkName, err := regexp.Compile(
		// Name 'a huge boar skull'
		`Name '([[:print:]]+)'`)
	ChkErr(err)
	chkKeyw, err := regexp.Compile(
		// Keyword 'skull boar', Item type: ARMOR
		`Keyword '([[:print:]]+)', Item type: ([[:word:]]+)`)
	ChkErr(err)
	chkWorn, err := regexp.Compile(
		// Item can be worn on:  HEAD
		`Item can be worn on: ([[:print:]]+)`)
	ChkErr(err)
	chkEff, err := regexp.Compile(
		// Item will give you following abilities:  NOBITS
		`Item will give you following abilities: ([[:print:]]+)`)
	ChkErr(err)
	chkFlag, err := regexp.Compile(
		// Item is: NOBITSNOBITS
		`Item is: ([[:print:]]+)`)
	ChkErr(err)
	chkRest, err := regexp.Compile(
		// NO-THIEF ANTI-ANTIPALADIN
		`[NO|ANTI]-`)
	ChkErr(err)
	chkWtval, err := regexp.Compile(
		// Weight: 2, Value: 0
		`Weight: ([[:digit:]]+), Value: ([[:digit:]]+)`)
	ChkErr(err)
	chkAC, err := regexp.Compile(
		// AC-apply is 8
		`AC-apply is ([[:digit:]]+)`)
	ChkErr(err)
	chkZone, err := regexp.Compile(
		// Zone: SP (UQ)
		`Zone: ([[:print:]]+)`)

	chkAttr, err := regexp.Compile(
		//     Affects : HITROLL By 2
		`Affects[ ]?: ([[:print:]]+) [B|b]y ([[:digit:]-]+)`)
	ChkErr(err)
	chkEnch, err := regexp.Compile(
		// Type: Holy     Damage: 100% Frequency: 100% Modifier: 0 Duration: 0 // enchantment
		`Type: ([[:print:]]+) Damage: ([[:digit:]]+)% ` +
			`Frequency: ([[:digit:]]+)[ ]?% ` +
			`Modifier: ([[:digit:]]+)[ ]{1,3}` +
			`Duration: ([[:digit:]]+)`)
	ChkErr(err)
	chkResis, err := regexp.Compile(
		// Resists: Fire : 5% Cold : 5% Elect : 5% Acid : 5% Poison: 5% Psi : 5%
		//     Unarmd:    2% Slash :    2% Bludgn:    2% Pierce:    2%
		//     Fire  :   10% Mental:    5%
		`([[:alpha:] ]{6}):[ ]{3,4}([[:digit:]-]{1,3})%[ ]?`)
	ChkErr(err)

	// item specials
	chkDice, err := regexp.Compile(
		// Damage Dice are '2D6' // old weapon dice
		`Damage (?:D|d)ice (?:are|is) '([[:digit:]D]+)'`)
	ChkErr(err)
	chkWeap, err := regexp.Compile(
		// Type: Morningstar Class: Simple // new weapon, type/class
		`Type: ([[:print:]]+) Class: ([[:print:]]+)`)
	ChkErr(err)
	chkCrit, err := regexp.Compile(
		// Damage:  2D5  Crit Range: 5%  Crit Bonus: 2x // new weapon, dice/crit/multi
		`Damage: [ ]?([[:digit:]D]+) [ ]?Crit Range: ([[:digit:]]+)% ` +
			`[ ]?Crit Bonus: ([[:digit:]]+)x`)
	ChkErr(err)
	chkPage, err := regexp.Compile(
		// Total Pages: 300 // spellbook
		`Total Pages: ([[:digit:]]+)`)
	ChkErr(err)
	chkPsp, err := regexp.Compile(
		// Has 700 capacity, charged with 700 points. // psp crystal
		`Has ([[:digit:]]+) capacity, charged with [[:digit:]]+ points.`)
	ChkErr(err)
	chkPois, err := regexp.Compile(
		// Poison affects as ray of enfeeblement at level 25. // type, level
		`Poison affects as ([[:print:]]+) at level ([[:digit:]]+).`)
	ChkErr(err)
	chkApps, err := regexp.Compile(
		// 1 applications remaining with 3 hits per application. // poison apps
		`([[:digit:]]+) applications remaining with ` +
			`([[:digit:]]+) hits per application.`)
	ChkErr(err)
	chkInstr, err := regexp.Compile(
		// Instrument Type: Drums, Quality: 8, Stutter: 7, Min Level: 1 // instrument
		`Instrument Type: ([[:print:]]+), Quality: ([[:digit:]]+), ` +
			`Stutter: ([[:digit:]]+), Min Level: ([[:digit:]]+)`)
	ChkErr(err)
	chkCharg, err := regexp.Compile(
		// Has 99 charges, with 99 charges left. // wand/staff
		`Has ([[:digit:]]+) charges, with ([[:digit:]]+) charges left.`)
	ChkErr(err)
	chkPot, err := regexp.Compile(
		// Level 35 spells of: // potion/scroll
		`Level ([[:digit:]]+) spells of: `)
	ChkErr(err)
	chkWand, err := regexp.Compile(
		// Level 1 spell of: Spell: airy water // staff/wand
		`Level ([[:digit:]]+) spell of: `)
	chkSpells, err := regexp.Compile(
		// Spell: protection from good // potion/scroll/wand/staff
		`Spell: ([[:lower:] ]+)`)
	ChkErr(err)
	chkCont, err := regexp.Compile(
		// Can hold 50 more lbs. // container
		`Can hold ([[:digit:]]+) more lbs\.`)
	ChkErr(err)
	chkWtless, err := regexp.Compile(
		// Can hold 600 more lbs with 300lbs weightless. // container
		`Can hold ([[:digit:]]+) more lbs with ([[:digit:]]+)lbs weightless.`)
	ChkErr(err)
	chkKey, err := regexp.Compile(
        // This key has a 0% chance to break when used. // key
        `This key has a ([[:digit:]]+)\% chance to break when used.`)
    ChkErr(err)

	for _, item := range items {
		// initialize item variables and slices
		full_stats, item_name, keywords, item_type := "", "", "", ""
		from_zone := "Unknown"
		weight, c_value := -1, -1
		var item_slots, item_effects, flags, item_flags, item_restricts []string
		var item_supps []string
		var item_attribs, item_specials, item_enchants, item_resists [][]string

		full_stats = item
		lines := strings.Split(item, "\n")
		var unmatch []string

		for _, line := range lines {
			switch {
			case chkName.MatchString(line):
				m = chkName.FindStringSubmatch(line)
				item_name = m[1]
			case chkKeyw.MatchString(line):
				m = chkKeyw.FindStringSubmatch(line)
				keywords = m[1]
				item_type = m[2]
			case chkWorn.MatchString(line):
				m = chkWorn.FindStringSubmatch(line)
				item_slots = strings.Fields(m[1])
			case chkEff.MatchString(line):
				m = chkEff.FindStringSubmatch(line)
				item_effects = strings.Fields(m[1])
			case chkFlag.MatchString(line):
				m = chkFlag.FindStringSubmatch(line)
				flags = strings.Fields(m[1])
				for _, flag := range flags {
					if chkRest.MatchString(flag) {
						item_restricts = append(item_restricts, flag)
					} else {
						item_flags = append(item_flags, flag)
					}
				}
			case chkWtval.MatchString(line):
				m = chkWtval.FindStringSubmatch(line)
				weight, err = strconv.Atoi(m[1])
				ChkErr(err)
				c_value, err = strconv.Atoi(m[2])
				ChkErr(err)
			case chkZone.MatchString(line):
				m = chkZone.FindStringSubmatch(line)
				if strings.Contains(m[1], " (") {
					ms := strings.Split(m[1], " (")
					from_zone = ms[0]
					ms[1] = strings.Trim(ms[1], ")")
					stuff := strings.Split(ms[1], "")
					for _, c := range stuff {
						item_supps = append(item_supps, c)
					}
				} else {
					from_zone = m[1]
				}
			case chkAttr.MatchString(line):
				m = chkAttr.FindStringSubmatch(line)
				item_attribs = append(item_attribs, []string{m[1], m[2]})
			case chkResis.MatchString(line):
				resis := chkResis.FindAllStringSubmatch(line, -1)
				for _, res := range resis {
					item_resists = append(item_resists, []string{
						strings.TrimSpace(res[1]), res[2]})
				}
			case chkEnch.MatchString(line):
				m = chkEnch.FindStringSubmatch(line)
				item_enchants = append(item_enchants,
					[]string{strings.TrimSpace(m[1]), m[2], m[3], m[4], m[5]})
			case chkAC.MatchString(line):
				m = chkAC.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "ac", m[1]})
			case chkDice.MatchString(line):
				m = chkDice.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "dice", m[1]})
			case chkWeap.MatchString(line):
				m = chkWeap.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "type", m[1]})
				item_specials = append(item_specials,
					[]string{item_type, "class", m[2]})
			case chkCrit.MatchString(line):
				m = chkCrit.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "dice", m[1]})
				item_specials = append(item_specials,
					[]string{item_type, "crit", m[2]})
				item_specials = append(item_specials,
					[]string{item_type, "multi", m[3]})
			case chkPsp.MatchString(line):
				m = chkPsp.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "psp", m[1]})
			case chkPage.MatchString(line):
				m = chkPage.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "pages", m[1]})
			case chkPois.MatchString(line):
				m = chkPois.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "type", m[1]})
				item_specials = append(item_specials,
					[]string{item_type, "level", m[2]})
			case chkApps.MatchString(line):
				m = chkApps.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "apps", m[1]})
				item_specials = append(item_specials,
					[]string{item_type, "hits", m[2]})
			case chkInstr.MatchString(line):
				m = chkInstr.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "type", m[1]})
				item_specials = append(item_specials,
					[]string{item_type, "quality", m[2]})
				item_specials = append(item_specials,
					[]string{item_type, "stutter", m[3]})
				item_specials = append(item_specials,
					[]string{item_type, "min_level", m[4]})
			case chkCharg.MatchString(line):
				m = chkCharg.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "charges", m[1]})
				//item_specials = append(item_specials,
				//	[]string{item_type, "cur_char", m[2]})
			case chkPot.MatchString(line):
				m = chkPot.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "level", m[1]})
				spells := chkSpells.FindAllStringSubmatch(line, -1)
				for n, spell := range spells {
					num := fmt.Sprintf("spell%d", n+1)
					item_specials = append(
						item_specials,
						[]string{
							item_type,
							num,
							strings.TrimSpace(spell[1]),
						},
					)
				}
			case chkWand.MatchString(line):
				m = chkWand.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "level", m[1]})
				m = chkSpells.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "spell", m[1]})
			case chkCont.MatchString(line):
				m = chkCont.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "holds", m[1]})
			case chkWtless.MatchString(line):
				m = chkWtless.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "holds", m[1]})
				item_specials = append(item_specials,
					[]string{item_type, "wtless", m[2]})
			case chkKey.MatchString(line):
				m = chkKey.FindStringSubmatch(line)
				item_specials = append(item_specials,
					[]string{item_type, "break", m[1]})
			default:
				unmatch = append(unmatch, line)
			}
		}
		// back to full item

		// translate from long form to abbreviated form
		// by building maps to match DB structure?
		db := OpenDB()
		defer db.Close()

		var a, b string
		query := "SELECT item_type, type_abbr FROM item_types"
		stmt, err := db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()
		rows, err := stmt.Query()
		ChkErr(err)
		defer rows.Close()
		types := make(map[string]string)
		for rows.Next() {
			err = rows.Scan(&a, &b)
			types[a] = b
		}
		ChkRows(rows)
		stmt.Close()
		query = "SELECT worn_slot, slot_abbr FROM slots"
		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()
		rows, err = stmt.Query()
		ChkErr(err)
		defer rows.Close()
		slots := make(map[string]string)
		for rows.Next() {
			err = rows.Scan(&a, &b)
			slots[a] = b
		}
		ChkRows(rows)
		stmt.Close()
		query = "SELECT effect_name, effect_abbr FROM effects"
		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()
		rows, err = stmt.Query()
		ChkErr(err)
		defer rows.Close()
		effs := make(map[string]string)
		for rows.Next() {
			err = rows.Scan(&a, &b)
			effs[a] = b
		}
		ChkRows(rows)
		stmt.Close()
		query = "SELECT flag_name, flag_abbr FROM flags"
		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()
		rows, err = stmt.Query()
		ChkErr(err)
		defer rows.Close()
		iflags := make(map[string]string)
		for rows.Next() {
			err = rows.Scan(&a, &b)
			iflags[a] = b
		}
		ChkRows(rows)
		stmt.Close()
		query = "SELECT restrict_name, restrict_abbr FROM restricts"
		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()
		rows, err = stmt.Query()
		ChkErr(err)
		defer rows.Close()
		restrs := make(map[string]string)
		for rows.Next() {
			err = rows.Scan(&a, &b)
			restrs[a] = b
		}
		ChkRows(rows)
		stmt.Close()
		query = "SELECT attrib_name, attrib_abbr FROM attribs"
		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()
		rows, err = stmt.Query()
		ChkErr(err)
		defer rows.Close()
		attrs := make(map[string]string)
		for rows.Next() {
			err = rows.Scan(&a, &b)
			attrs[a] = b
		}
		ChkRows(rows)
		stmt.Close()
		query = "SELECT resist_name, resist_abbr FROM resists"
		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()
		rows, err = stmt.Query()
		ChkErr(err)
		defer rows.Close()
		resis := make(map[string]string)
		for rows.Next() {
			err = rows.Scan(&a, &b)
			resis[a] = b
		}
		ChkRows(rows)
		stmt.Close()
		query = "SELECT supp_value, supp_abbr FROM supps"
		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()
		rows, err = stmt.Query()
		ChkErr(err)
		defer rows.Close()
		supps := make(map[string]string)
		for rows.Next() {
			err = rows.Scan(&a, &b)
			supps[a] = b
		}
		ChkRows(rows)
		stmt.Close()

		loc, err := time.LoadLocation("America/New_York")
		ChkErr(err)
		date := time.Now().In(loc).Format("2006-01-02")
		// when using time.Format(), you need to make the format text
		// from this exact datetime: Mon Jan 2 15:04:05 -0700 MST 2006

		// check if exact name, keywords, type, weight, value is already in DB
		query = "SELECT item_id, short_stats " +
			"FROM items WHERE item_name = ? " +
			"AND keywords = ? AND item_type = ? " +
			"AND weight = ? AND c_value = ?"
		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		var id int64
		var short_stats string
		err = stmt.QueryRow(
			item_name, keywords, types[item_type], weight, c_value,
		).Scan(&id, &short_stats)

		if err == sql.ErrNoRows {
			// if it's not in the DB, check for existing items
			query = "SELECT item_id, long_stats " +
				"FROM items WHERE item_name = ?"
			stmt, err = db.Prepare(query)
			ChkErr(err)
			defer stmt.Close()

			err = stmt.QueryRow(item_name).Scan(&id, &short_stats)
			if err == sql.ErrNoRows {
				log.Print(FindItem(item_name, "long_stats"))
			} else if err != nil {
				log.Fatal(err)
			} else {
				log.Printf("Name match: item_id=%d; %s", id, short_stats)
			}

			//compile full insert queries
			tx, err := db.Begin()
			ChkErr(err)

			query = "INSERT INTO items " +
				"(item_name, keywords, weight, c_value, " +
				"item_type, last_id, from_zone, full_stats) " +
				"VALUES(?, ?, ?, ?, ?, ?, ?, ?);"
			stmt, err := tx.Prepare(query)
			ChkErr(err)
			defer stmt.Close()

			res, err := stmt.Exec(
				item_name, keywords, weight, c_value,
				types[item_type], date, from_zone, full_stats)
			ChkErr(err)

			id, err = res.LastInsertId()
			ChkErr(err)
			inserted++
			for _, um := range unmatch {
				if !strings.Contains(um, "Can affect you as :") &&
					!strings.Contains(um, "Enchantments:") &&
					!strings.Contains(um, "You feel informed:") {
					log.Printf("Unmatched: %s", um)
				}
			}
			sqls := fmt.Sprintf("Inserted new item: id[%d], name: %s\n",
				id, item_name)
			sqls += "----------------------------*/\n"
			query = "INSERT INTO items " +
				"(item_id, item_name, keywords, weight, c_value, " +
				"item_type, last_id, from_zone, full_stats) " +
				"VALUES(%d, %s, %s, %d, %d, %s, %s, %s, %s);"
			sqls += fmt.Sprintf(query+"\n", id,
				strconv.Quote(item_name),
				strconv.Quote(keywords),
				weight, c_value,
				strconv.Quote(types[item_type]),
				strconv.Quote(date),
				strconv.Quote(from_zone),
				strconv.Quote(full_stats),
			)

			for _, slot := range item_slots {
				query = "INSERT INTO item_slots VALUES(?, ?);"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, slots[slot])
				ChkErr(err)
				query = strings.Replace(query, "?", "%v", -1)
				sqls += fmt.Sprintf(query+"\n", id,
					strconv.Quote(slots[slot]))
			}
			for _, eff := range item_effects {
				if eff != "NOBITS" && eff != "GROUP_CACHED" {
					query = "INSERT INTO item_effects VALUES(?, ?);"
					stmt, err := tx.Prepare(query)
					ChkErr(err)
					defer stmt.Close()

					_, err = stmt.Exec(id, effs[eff])
					ChkErr(err)
					query = strings.Replace(query, "?", "%v", -1)
					sqls += fmt.Sprintf(query+"\n", id,
						strconv.Quote(effs[eff]))
				}
			}
			for _, flag := range item_flags {
				if flag != "NOBITS" && flag != "NOBITSNOBITS" {
					query = "INSERT INTO item_flags VALUES(?, ?);"
					stmt, err := tx.Prepare(query)
					ChkErr(err)
					defer stmt.Close()

					_, err = stmt.Exec(id, iflags[flag])
					ChkErr(err)
					query = strings.Replace(query, "?", "%v", -1)
					sqls += fmt.Sprintf(query+"\n", id,
						strconv.Quote(iflags[flag]))
				}
			}
			for _, rest := range item_restricts {
				query = "INSERT INTO item_restricts VALUES(?, ?);"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, restrs[rest])
				ChkErr(err)
				query = strings.Replace(query, "?", "%v", -1)
				sqls += fmt.Sprintf(query+"\n", id,
					strconv.Quote(restrs[rest]))
			}
			for _, attr := range item_attribs {
				query = "INSERT INTO item_attribs VALUES(?, ?, ?);"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, attrs[attr[0]], attr[1])
				ChkErr(err)
				query = strings.Replace(query, "?", "%v", -1)
				sqls += fmt.Sprintf(query+"\n", id,
					strconv.Quote(attrs[attr[0]]), attr[1])
			}
			for _, spec := range item_specials {
				query = "INSERT INTO item_specials VALUES(?, ?, ?, ?);"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, types[spec[0]], spec[1], spec[2])
				ChkErr(err)
				query = strings.Replace(query, "?", "%v", -1)
				sqls += fmt.Sprintf(query+"\n", id,
					strconv.Quote(types[spec[0]]),
					strconv.Quote(spec[1]),
					strconv.Quote(spec[2]),
				)
			}
			for _, ench := range item_enchants {
				query = "INSERT INTO item_enchants VALUES(?, ?, ?, ?, ?, ?);"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id,
					ench[0], ench[1], ench[2], ench[3], ench[4])
				ChkErr(err)
				query = strings.Replace(query, "?", "%v", -1)
				sqls += fmt.Sprintf(query+"\n", id,
					strconv.Quote(ench[0]),
					ench[1], ench[2], ench[3], ench[4],
				)
			}
			for _, res := range item_resists {
				query = "INSERT INTO item_resists VALUES(?, ?, ?);"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, resis[res[0]], res[1])
				ChkErr(err)
				query = strings.Replace(query, "?", "%v", -1)
				sqls += fmt.Sprintf(query+"\n", id,
					strconv.Quote(resis[res[0]]), res[1])
			}
			for _, supp := range item_supps {
				query = "INSERT INTO item_supps VALUES(?, ?);"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, supps[supp])
				ChkErr(err)
				query = strings.Replace(query, "?", "%v", -1)
				sqls += fmt.Sprintf(query+"\n", id,
					strconv.Quote(supps[supp]))
			}
			tx.Commit()
			if from_zone == "Unknown" {
				sqls += fmt.Sprintf(
					"--UPDATE items SET from_zone = \"?\" "+
						"WHERE item_id = %d;\n", id)
			}
			sqls += fmt.Sprintf(
				"--INSERT INTO item_procs (item_id, proc_name) "+
					"VALUES(%d, \"?\");\n", id)
			sqls += "/*----------------------------\n"
			log.Print(sqls)
		} else if err != nil {
			log.Fatal(err)
		} else {
			ignored++
			for _, um := range unmatch {
				if !strings.Contains(um, "Can affect you as :") &&
					!strings.Contains(um, "Enchantments:") &&
					!strings.Contains(um, "You feel informed:") {
					log.Printf("Unmatched: %s", um)
				}
			}
			sqls := fmt.Sprintf("Item already exists: item_id=%d; name: %s\n",
				id, item_name)
			sqls += "----------------------------*/\n"
			sqls += fmt.Sprintf(
				"UPDATE items SET last_id = '%s' WHERE item_id = %d;\n",
				date, id)
			sqls += fmt.Sprintf("UPDATE items SET full_stats = %s "+
				"WHERE item_id = %d;\n",
				strconv.Quote(full_stats), id)
			for _, slot := range item_slots {
				sqls += fmt.Sprintf(
					"--INSERT INTO item_slots VALUES(%d, %s);\n",
					id, strconv.Quote(slots[slot]))
			}
			for _, eff := range item_effects {
				if eff != "NOBITS" && eff != "GROUP_CACHED" {
					sqls += fmt.Sprintf(
						"--INSERT INTO item_effects VALUES(%d, %s);\n",
						id, strconv.Quote(effs[eff]))
				}
			}
			for _, flag := range item_flags {
				if flag != "NOBITS" && flag != "NOBITSNOBITS" {
					sqls += fmt.Sprintf(
						"--INSERT INTO item_flags VALUES(%d, %s);\n",
						id, strconv.Quote(iflags[flag]))
				}
			}
			for _, rest := range item_restricts {
				sqls += fmt.Sprintf(
					"--INSERT INTO item_restricts VALUES(%d, %s);\n",
					id, strconv.Quote(restrs[rest]))
			}
			for _, attr := range item_attribs {
				sqls += fmt.Sprintf(
					"--INSERT INTO item_attribs VALUES(%d, %s, %s);\n",
					id, strconv.Quote(attrs[attr[0]]), attr[1])
			}
			for _, spec := range item_specials {
				sqls += fmt.Sprintf(
					"--INSERT INTO item_specials VALUES(%d, %s, %s, %s);\n",
					id, strconv.Quote(types[spec[0]]),
					strconv.Quote(spec[1]),
					strconv.Quote(spec[2]),
				)
			}
			for _, ench := range item_enchants {
				sqls += fmt.Sprintf(
					"--INSERT INTO item_enchants "+
						"VALUES(%d, %s, %s, %s, %s, %s);\n",
					id, strconv.Quote(ench[0]),
					ench[1], ench[2], ench[3], ench[4],
				)
			}
			for _, res := range item_resists {
				sqls += fmt.Sprintf(
					"--INSERT INTO item_resists VALUES(%d, %s, %s);\n",
					id, strconv.Quote(resis[res[0]]), res[1])
			}
			for _, supp := range item_supps {
				sqls += fmt.Sprintf(
					"--INSERT INTO item_supps VALUES(%d, %s);\n",
					id, strconv.Quote(supps[supp]))
			}
			if from_zone != "Unknown" {
				sqls += fmt.Sprintf(
					"--UPDATE items SET from_zone = %s "+
						"WHERE item_id = %d;\n", strconv.Quote(from_zone), id)
			}
			sqls += fmt.Sprintf(
				"--INSERT INTO item_procs (item_id, proc_name) "+
					"VALUES(%d, \"?\");\n", id)

			sqls += "/*----------------------------\n"
			log.Print(sqls)
			log.Println(short_stats)
			// manually compare full_stats vs. short_stats for possible update
		}
		// send all insert/update queries as a .sql file to review
		// manual updates: procs, zone, supps
		// end of DB stuff */
	}
	txt := []string{
		fmt.Sprintf("Items Inserted: %d, Items Ignored: %d\n",
			inserted, ignored)}
	return txt
}
