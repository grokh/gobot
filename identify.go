package main

import (
	//"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	//"log"
	"regexp"
	"strconv"
	"strings"
	//"time"
)

func Identify(filename string) {
	content, err := ioutil.ReadFile(filename)
	ChkErr(err)
	text := string(content)

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
		`\n(    [[:alpha:] ]{6}:[[:blank:]]{3,4}[[:digit:]]{1,2}% )`)
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
	chkKey, err := regexp.Compile(
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

	chkAttr, err := regexp.Compile(
		//     Affects : HITROLL By 2
		`Affects : ([[:print:]]+) [B|b]y ([[:digit:]-]+)`)
	ChkErr(err)
	chkEnch, err := regexp.Compile(
		// Type: Holy     Damage: 100% Frequency: 100% Modifier: 0 Duration: 0 // enchantment
		`Type: ([[:print:]]+) Damage: ([[:digit:]]+)% ` +
			`Frequency: ([[:digit:]]+)[ ]?% ` +
			`Modifier: ([[:digit:]]+) ` +
			`Duration: ([[:digit:]]+)`)
	ChkErr(err)
	chkResis, err := regexp.Compile(
		// Resists: Fire : 5% Cold : 5% Elect : 5% Acid : 5% Poison: 5% Psi : 5%
		//     Unarmd:    2% Slash :    2% Bludgn:    2% Pierce:    2% 
		//     Fire  :   10% Mental:    5% 
		`([[:alpha:] ]{6}):[ ]{3,4}([[:digit:]]{1,2})% `)
	ChkErr(err)

	// item specials
	chkDice, err := regexp.Compile(
		// Damage Dice are '2D6' // old weapon dice
		`Damage Dice are '([[:digit:]D]+)'`)
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
		`Can hold ([[:digit:]]+) more lbs.`)
	ChkErr(err)
	chkWtless, err := regexp.Compile(
		// Can hold 600 more lbs with 300lbs weightless. // container
		`Can hold ([[:digit:]]+) more lbs with ([[:digit:]]+)lbs weightless.`)
	ChkErr(err)

	for _, item := range items {
		// initialize item variables and slices
		full_stats, item_name, keywords, item_type := "", "", "", ""
		weight, c_value := -1, -1
		var item_slots, item_effects, flags, item_flags, item_restricts []string
		var item_attribs, item_specials, item_enchants, item_resists [][]string

		full_stats = item
		lines := strings.Split(item, "\n")
		var unmatch []string

		for _, line := range lines {
			switch {
			case chkName.MatchString(line):
				m = chkName.FindStringSubmatch(line)
				item_name = m[1]
			case chkKey.MatchString(line):
				m = chkKey.FindStringSubmatch(line)
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
					item_specials = append(item_specials,
						[]string{item_type, num, spell[1]})
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
			default:
				unmatch = append(unmatch, line)
			}
		}
		// back to full item
		// translate from long form to abbreviated form
		// or do that up above?
		fmt.Printf("Name: %s\nKeywords: %s\nType: %s\n",
			item_name, keywords, item_type)
		fmt.Printf("Weight: %d\nValue: %d\n", weight, c_value)
		for _, slot := range item_slots {
			fmt.Printf("Slot: %s\n", slot)
		}
		for _, eff := range item_effects {
			if eff != "NOBITS" && eff != "GROUP_CACHED" {
				fmt.Printf("Effect: %s\n", eff)
			}
		}
		for _, flag := range item_flags {
			if flag != "NOBITS" && flag != "NOBITSNOBITS" {
				fmt.Printf("Flag: %s\n", flag)
			}
		}
		for _, rest := range item_restricts {
			fmt.Printf("Restrict: %s\n", rest)
		}
		for _, attr := range item_attribs {
			fmt.Printf("Attrib: %s, Value: %s\n", attr[0], attr[1])
		}
		for _, spec := range item_specials {
			fmt.Printf("Special: Type: %s, Abbr: %s, Value: %s\n",
				spec[0], spec[1], spec[2])
		}
		for _, ench := range item_enchants {
			fmt.Printf("Enchant: Name: %s, Dam_Pct: %s, Freq_Pct: %s, "+
				"Sv_Mod: %s, Duration: %s\n",
				ench[0], ench[1], ench[2], ench[3], ench[4])
		}
		for _, res := range item_resists {
			fmt.Printf("Resist: Name: %s, Value: %s\n", res[0], res[1])
		}
		for _, um := range unmatch {
			if !strings.Contains(um, "Can affect you as :") &&
				!strings.Contains(um, "Enchantments:") &&
				!strings.Contains(um, "You feel informed:") {
				fmt.Println("Unmatched: ", um)
			}
		}
		_ = full_stats
		fmt.Print("\n----------\n\n")
/*
		loc, err := time.LoadLocation("America/New_York")
		ChkErr(err)
		date := time.Now().In(loc)

		db := OpenDB()
		defer db.Close()

		// check if exact name is already in DB
		query := "SELECT item_id FROM items WHERE item_name = ? " +
			"AND keywords = ? AND item_type = ? " +
			"AND weight = ? AND c_value = ?"
		stmt, err := db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		var id int64
		err = stmt.QueryRow(
			item_name, keywords, item_type, weight, c_value,
		).Scan(&id)

		if err == sql.ErrNoRows {
			// if it's not in the DB, compile full insert queries
			tx, err := db.Begin()
			ChkErr(err)

			query = "INSERT INTO items "+
				"(item_name, keywords, weight, c_value, "+
				"item_type, full_stats, last_id) "+
				"VALUES(?, ?, ?, ?, ?, ?, ?)"
			stmt, err := tx.Prepare(query)
			ChkErr(err)
			defer stmt.Close()

			res, err := stmt.Exec(
				item_name, keywords, weight, c_value,
				item_type, full_stats, date)
			ChkErr(err)

			id, err = res.LastInsertId()
			ChkErr(err)

			for _, slot := range item_slots {
				query = "INSERT INTO item_slots VALUES(?, ?)"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, slot)
				ChkErr(err)
			}
			for _, eff := range item_effects {
				if eff != "NOBITS" && eff != "GROUP_CACHED" {
					query = "INSERT INTO item_effects VALUES(?, ?)"
					stmt, err := tx.Prepare(query)
					ChkErr(err)
					defer stmt.Close()

					_, err = stmt.Exec(id, eff)
					ChkErr(err)
				}
			}
			for _, flag := range item_flags {
				if flag != "NOBITS" && flag != "NOBITSNOBITS" {
					query = "INSERT INTO item_flags VALUES(?, ?)"
					stmt, err := tx.Prepare(query)
					ChkErr(err)
					defer stmt.Close()

					_, err = stmt.Exec(id, flag)
					ChkErr(err)
				}
			}
			for _, rest := range item_restricts {
				query = "INSERT INTO item_restricts VALUES(?, ?)"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, rest)
				ChkErr(err)
			}
			for _, attr := range item_attribs {
				query = "INSERT INTO item_attribs VALUES(?, ?, ?)"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, attr[0], attr[1])
				ChkErr(err)
			}
			for _, spec := range item_specials {
				query = "INSERT INTO item_specials VALUES(?, ?, ?, ?)"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, spec[0], spec[1], spec[2])
				ChkErr(err)
			}
			for _, ench := range item_enchants {
				query = "INSERT INTO item_enchants VALUES(?, ?, ?, ?, ?, ?)"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, ench[0], ench[1], ench[2], ench[3], ench[4])
				ChkErr(err)
			}
			for _, res := range item_resists {
				query = "INSERT INTO item_resists VALUES(?, ?, ?)"
				stmt, err := tx.Prepare(query)
				ChkErr(err)
				defer stmt.Close()

				_, err = stmt.Exec(id, res[0], res[1])
				ChkErr(err)
			}
			tx.Commit()
		} else if err != nil {
			log.Fatal(err)
		} else {
			// if same name and such, update the date of last_id
			// manually compare full_stats vs. short_stats for possible update
		}
		// send all insert/update queries as a .sql file to review
		// manual updates: procs, zone, supps
		// end of DB stuff */
	}
}
