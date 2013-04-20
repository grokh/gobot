package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	//"log"
	"strings"
)

var i struct {
	id, name, itype, zone, date, keys, s     string // base
	wt, val                                  int    // base
	specs, procs, enchs, flags, restr, zones string // temp
	tmp, tmpb, txt                           string // temp
	tmp1, tmp2, tmp3, tmp4                   int    // temp
	txt1, txt2, txt3, txt4, txt5             string // temp, special ordering
	ids, stats                               []interface{}
}

func ShortStats() {
	db, err := sql.Open("sqlite3", "toril.db")
	ChkErr(err)
	defer db.Close()

	query := "SELECT item_id FROM items"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err := stmt.Query()
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.id)

		// collect general item information
		query = "SELECT item_name, item_type, weight, c_value, " +
			"from_zone, last_id " +
			"FROM items WHERE item_id = ?"
		stmt, err := db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		err = stmt.QueryRow(i.id).Scan(
			&i.name, &i.itype, &i.wt, &i.val,
			&i.zone, &i.date,
		)
		ChkErr(err)
		i.s = i.name

		// collect slots (i.slots, i.slot)
		query = "SELECT slot_abbr FROM item_slots WHERE item_id = ?"
		stmt2, err := db.Prepare(query)
		ChkErr(err)
		defer stmt2.Close()

		rows2, err := stmt2.Query(i.id)
		ChkErr(err)
		defer rows2.Close()

		for rows2.Next() {
			err = rows2.Scan(&i.tmp)
			i.s += " (" + strings.Title(i.tmp) + ")"
		}

		// collect armor class (i.spec, but only for armor)
		query = "SELECT spec_value FROM item_specials " +
			"WHERE item_id = ? AND spec_abbr = 'ac'"
		stmt3, err := db.Prepare(query)
		ChkErr(err)
		defer stmt3.Close()

		rows3, err := stmt3.Query(i.id)
		ChkErr(err)
		defer rows3.Close()

		for rows3.Next() {
			err = rows3.Scan(&i.tmp1)
			i.s += fmt.Sprintf(" AC:%d", i.tmp1)
		}

		// collect attributes (i.attrs, i.attr)
		query = "SELECT attrib_abbr, attrib_value " +
			"FROM item_attribs WHERE item_id = ?"
		stmt4, err := db.Prepare(query)
		ChkErr(err)
		defer stmt4.Close()

		rows4, err := stmt4.Query(i.id)
		ChkErr(err)
		defer rows4.Close()

		for rows4.Next() {
			err = rows4.Scan(&i.tmp, &i.tmp1)
			i.s += fmt.Sprintf(" %s:%d", strings.Title(i.tmp), i.tmp1)
		}

		// collect resistances (i.resis, i.res)
		query = "SELECT resist_abbr, resist_value " +
			"FROM item_resists WHERE item_id = ?"
		stmt5, err := db.Prepare(query)
		ChkErr(err)
		defer stmt5.Close()

		rows5, err := stmt5.Query(i.id)
		ChkErr(err)
		defer rows5.Close()

		for rows5.Next() {
			err = rows5.Scan(&i.tmp, &i.tmp1)
			i.s += fmt.Sprintf(" %s:%d%%", strings.Title(i.tmp), i.tmp1)
		}

		// collect item effects (i.effs, i.eff)
		query = "SELECT effect_abbr FROM item_effects WHERE item_id = ?"
		stmt6, err := db.Prepare(query)
		ChkErr(err)
		defer stmt6.Close()

		rows6, err := stmt6.Query(i.id)
		ChkErr(err)
		defer rows6.Close()

		for rows6.Next() {
			err = rows6.Scan(&i.tmp)
			if strings.Contains(i.tmp, "_") {
				s := strings.Split(i.tmp, "_")
				for n, v := range s {
					s[n] = strings.Title(v)
				}
				i.tmp = strings.Join(s, "_")
				i.s += " " + i.tmp
			} else {
				i.s += " " + strings.Title(i.tmp)
			}
		}

		// collect specials (i.specs, i.spec) and break them down by type
		i.specs = " *"
		i.txt1, i.txt2, i.txt3, i.txt4, i.txt5 = "", "", "", "", ""
		query = "SELECT item_type, spec_abbr, spec_value " +
			"FROM item_specials WHERE item_id = ? AND spec_abbr != 'ac'"
		stmt7, err := db.Prepare(query)
		ChkErr(err)
		defer stmt7.Close()

		rows7, err := stmt7.Query(i.id)
		ChkErr(err)
		defer rows7.Close()

		for rows7.Next() {
			err = rows7.Scan(&i.txt, &i.tmp, &i.tmpb)
			if !strings.Contains(i.specs, "(") {
				i.specs += " (" + strings.Title(i.txt) + ")"
			}
			switch {
			case i.txt == "crystal" || i.txt == "spellbook" ||
				i.txt == "comp_bag" || i.txt == "ammo":
				i.txt1 += " " + strings.Title(i.tmp) + ":" + i.tmpb
			case i.txt == "container":
				if i.tmp == "holds" {
					i.txt1 += " Holds:" + i.tmpb
				} else if i.tmp == "wtless" {
					i.txt2 += " Wtless:" + i.tmpb
				}
			case i.txt == "poison":
				if i.tmp == "level" {
					i.txt1 += " Lvl:" + i.tmpb
				} else if i.tmp == "type" {
					i.txt2 += " Type:" + i.tmpb
				} else if i.tmp == "apps" {
					i.txt3 += " Apps:" + i.tmpb
				}
			case i.txt == "scroll" || i.txt == "potion":
				if i.tmp == "level" {
					i.txt1 += " Lvl:" + i.tmpb
				} else if i.tmp == "spell1" {
					i.txt2 += " " + i.tmpb
				} else if i.tmp == "spell2" {
					i.txt3 += " - " + i.tmpb
				} else if i.tmp == "spell3" {
					i.txt4 += " - " + i.tmpb
				}
			case i.txt == "staff" || i.txt == "wand":
				if i.tmp == "level" {
					i.txt1 += " Lvl:" + i.tmpb
				} else if i.tmp == "spell" {
					i.txt2 += " " + i.tmpb
				} else if i.tmp == "charges" {
					i.txt3 += " Charges:" + i.tmpb
				}
			case i.txt == "instrument":
				if i.tmp == "quality" {
					i.txt1 += " Quality:" + i.tmpb
				} else if i.tmp == "stutter" {
					i.txt2 += " Stuter:" + i.tmpb
				} else if i.tmp == "min_level" {
					i.txt3 += " Min_Level:" + i.tmpb
				}
			case i.txt == "weapon":
				if i.tmp == "dice" {
					i.txt1 += " Dice:" + i.tmpb
				} else if i.tmp == "crit" {
					i.txt2 += " Crit:" + i.tmpb + "%"
				} else if i.tmp == "multi" {
					i.txt3 += " Multi:" + i.tmpb + "x"
				} else if i.tmp == "class" {
					i.txt4 += " (Class: " + i.tmpb + ","
				} else if i.tmp == "type" {
					i.txt5 += " Type: " + i.tmpb + ")"
				}
			}
		}
		i.specs += i.txt1 + i.txt2 + i.txt3 + i.txt4 + i.txt5
		if i.specs != " *" {
			i.s += i.specs
		}

		// collect procs (i.procs, i.proc)
		i.procs = " *"
		query = "SELECT proc_name FROM item_procs WHERE item_id = ?"
		stmt8, err := db.Prepare(query)
		ChkErr(err)
		defer stmt8.Close()

		rows8, err := stmt8.Query(i.id)
		ChkErr(err)
		defer rows8.Close()

		for rows8.Next() {
			err = rows8.Scan(&i.tmp)
			if i.procs == " *" {
				i.procs += " Procs: " + i.tmp
			} else {
				i.procs += " - " + i.tmp
			}
		}
		if i.procs != " *" {
			i.s += i.procs
		}

		// collect enchantments (i.enchs, i.ench)
		i.enchs = " *"
		query = "SELECT ench_name, dam_pct, freq_pct, sv_mod, duration " +
			"FROM item_enchants WHERE item_id = ?"
		stmt9, err := db.Prepare(query)
		ChkErr(err)
		defer stmt9.Close()

		rows9, err := stmt9.Query(i.id)
		ChkErr(err)
		defer rows9.Close()

		for rows9.Next() {
			err = rows9.Scan(&i.tmp, &i.tmp1, &i.tmp2, &i.tmp3, &i.tmp4)
			if i.enchs != " *" {
				i.enchs += " -"
			}
			i.enchs += fmt.Sprintf(" %s %d%% %d%% %d %d",
				strings.Title(i.tmp), i.tmp1, i.tmp2, i.tmp3, i.tmp4)
		}
		if i.enchs != " *" {
			i.s += i.enchs
		}

		// collect item flags (i.flags, i.flag)
		i.flags = " *"
		query = "SELECT flag_abbr FROM item_flags WHERE item_id = ?"
		stmt10, err := db.Prepare(query)
		ChkErr(err)
		defer stmt10.Close()

		rows10, err := stmt10.Query(i.id)
		ChkErr(err)
		defer rows10.Close()

		for rows10.Next() {
			err = rows10.Scan(&i.tmp)
			if strings.Contains(i.tmp, "_") {
				s := strings.Split(i.tmp, "_")
				for n, v := range s {
					s[n] = strings.Title(v)
				}
				i.tmp = strings.Join(s, "_")
				i.flags += " " + i.tmp
			} else {
				i.flags += " " + strings.Title(i.tmp)
			}
		}
		if i.flags != " *" {
			i.s += i.flags
		}

		// collect restrictions (i.restr, i.rest)
		i.restr = " *"
		query = "SELECT restrict_abbr FROM item_restricts WHERE item_id = ?"
		stmt11, err := db.Prepare(query)
		ChkErr(err)
		defer stmt11.Close()

		rows11, err := stmt11.Query(i.id)
		ChkErr(err)
		defer rows11.Close()

		for rows11.Next() {
			err = rows11.Scan(&i.tmp)
			i.restr += " " + strings.Title(i.tmp)
		}
		if i.restr != " *" && i.flags == " *" {
			i.s += i.restr
		} else if i.restr != " *" && i.flags != " *" {
			i.restr = i.restr[:1] + i.restr[3:]
			i.s += i.restr
		}

		// collect item supplementals (i.supps, i.supp)
		query = "SELECT supp_value FROM supps s, item_supps i " +
			"WHERE s.supp_abbr = i.supp_abbr AND item_id = ?"
		stmt12, err := db.Prepare(query)
		ChkErr(err)
		defer stmt12.Close()

		rows12, err := stmt12.Query(i.id)
		ChkErr(err)
		defer rows12.Close()

		// put in misc info
		i.itype = " *"
		if i.wt != -1 {
			i.itype += fmt.Sprintf(" Wt:%d", i.wt)
		}
		if i.val != -1 {
			i.itype += fmt.Sprintf(" Val:%d", i.val)
		}

		// construct the zone and last id
		i.zones = ""
		for rows12.Next() {
			err = rows12.Scan(&i.tmp)
			if i.tmp != "NoID" {
				i.zones += i.tmp
			} else {
				i.itype += " " + i.tmp
			}
		}
		if i.zones != "" {
			i.zone += " (" + i.zones + ")"
		}
		if i.itype != " *" {
			i.s += i.itype
		}
		i.s += " * Zone: " + i.zone + " * Last ID: " + i.date

		// debugging
		//log.Println(i.s)

		// save the short_stats and id for later use
		i.ids = append(i.ids, i.id)
		i.stats = append(i.stats, i.s)
	}
	err = rows.Err()
	ChkErr(err)
	rows.Close()

	// put the batched short_stats into the database
	tx, err := db.Begin()
	ChkErr(err)
	stmt, err = tx.Prepare("UPDATE items SET short_stats = ? WHERE item_id = ?")
	ChkErr(err)
	defer stmt.Close()
	if len(i.ids) == len(i.stats) {
		for n := 0; n < len(i.ids); n++ {
			_, err = stmt.Exec(i.stats[n], i.ids[n])
			ChkErr(err)
		}
	}
	tx.Commit()
}

func LongStats() {
	db, err := sql.Open("sqlite3", "toril.db")
	ChkErr(err)
	defer db.Close()

	query := "SELECT item_id FROM items"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	rows, err := stmt.Query()
	ChkErr(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&i.id)

		// collect general item information
		query = "SELECT item_name, item_type, weight, c_value, " +
			"zone_name, last_id, keywords " +
			"FROM items i, zones z " +
			"WHERE i.from_zone = z.zone_abbr AND item_id = ?"
		stmt, err := db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		err = stmt.QueryRow(i.id).Scan(
			&i.name, &i.itype, &i.wt, &i.val,
			&i.zone, &i.date, &i.keys,
		)
		ChkErr(err)

		// collect slots (i.slots, i.slot)
		// collect armor class (specials, but only for armor)
		// collect attributes (i.attrs, i.attr)
		// collect resistances (i.resis, i.res)
		// collect item effects (i.effs, i.eff)
		// collect specials (i.specs, i.spec) and break them down by type
		// collect procs (i.procs, i.proc)
		// collect enchantments (i.enchs, i.ench)
		// collect item flags (i.flags, i.flag)
		// collect restrictions (i.restr, i.rest)
		// collect item supplementals (i.supps, i.supp)
		// construct the remainder of long_stats
	}
	err = rows.Err()
	ChkErr(err)
	rows.Close()
}
