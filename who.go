package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
)

func Who(char string, lvl int) {
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
	var name string
	err = stmt.QueryRow(char).Scan(&acc, &name)
	if err != nil {
		// if char doesn't exist, 'who char'
		fmt.Printf("who %s\n", char)
		return
	} else {
		// if char does exist, tell the DB the time they were spotted and
		// update their level 
		// todo: also check class change for necro->lich
		// todo: also check for account name change
		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}
		stmt, err := tx.Prepare("UPDATE chars SET char_level = ?, last_seen = ? WHERE account_name = ? AND char_name = ?")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(lvl, date, acc, name)
		if err != nil {
			log.Fatal(err)
		}
		tx.Commit()
	}
}

func WhoChar(char string, lvl int, class string, race string, acct string) {
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
        var name string
        err = stmt.QueryRow(char).Scan(&acc, &name)
        if err != nil {
		// if no char, check if account exists in DB, create char
	}
}
