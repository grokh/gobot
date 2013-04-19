package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	//"strings"
)

var Item struct {
	id      string
	name    string
	itype   string
	zonext  string
	last_id string
	slots   []string
	specs   []string
	attrs   []string
	resi    []string
	effects []string
	enchs   []string
	flags   []string
	restr   []string
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
		err = rows.Scan(&Item.id)
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
		err = rows.Scan(&Item.id)
	}
	err = rows.Err()
	ChkErr(err)
	rows.Close()
}
