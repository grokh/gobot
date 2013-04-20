package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	//"strings"
)

var i struct {
	id, name, itype, wt, val, zone, zonext, date, s                string
	slots, specs, attr, resis, effects, enchs, flags, restr, supps []string
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
		log.Printf("Name: %s\n", i.name)
	}
	err = rows.Err()
	ChkErr(err)
	rows.Close()
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
	}
	err = rows.Err()
	ChkErr(err)
	rows.Close()
}
