package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

func Who(lvl int, name string) {
	date := time.Now()
	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	// check if character exists in DB
	stmt, err := db.Prepare("SELECT account_name, char_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?)")
	if err != nil {
		fmt.Println(err)
		return
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
		tx, err := db.Begin()
		if err != nil {
			fmt.Println(err)
			return
		}
		stmt, err := tx.Prepare("UPDATE chars SET char_level = ?, last_seen = ? " +
			"WHERE account_name = ? AND char_name = ?")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(lvl, date, acc, char)
		if err != nil {
			fmt.Println(err)
			return
		}
		tx.Commit()
	}
}
