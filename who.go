package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
)

func Who(lvl int, name string) {
	date := time.Now()
	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// check if character exists in DB
	stmt, err := db.Prepare("SELECT account_name, char_name FROM chars WHERE LOWER(char_name) = LOWER(?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var acc string
	var char string
	err = stmt.QueryRow(name).Scan(&acc, &char)
	if err != nil {
		// if char doesn't exist, 'who char'
		fmt.Printf("who %s\n", name)
		return
	} else {
		// if char does exist, tell the DB the time they were spotted and
		// update their level 
		// todo: also class change for necro->lich
		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}
		stmt, err := tx.Prepare("UPDATE chars SET char_level = ?, last_seen = ? WHERE account_name = ? AND char_name = ?")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(lvl, date, acc, char)
		if err != nil {
			log.Fatal(err)
		}
		tx.Commit()
	}
}
