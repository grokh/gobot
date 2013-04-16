package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
	"strings"
)

func WhoBatch(batch string) {
	who := strings.Split(batch, "|")
	// regex for who line :/
}

func Who(char string, lvl int) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}
	date := time.Now().In(loc)
	// debugging
	fmt.Printf("Time: %v\n", date)

	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// TODO: switch to update first, who on particular err
	// check if character exists in DB
	stmt, err := db.Prepare("SELECT account_name, char_name FROM chars WHERE LOWER(char_name) = LOWER(?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var acc string
	var name string
	err = stmt.QueryRow(char).Scan(&acc, &name)
	if err != nil { // change to if err == actual NoData err
		// if char doesn't exist, 'who char'
		fmt.Printf("who %s\n", char)
		return
	} else if err != nil {
		log.Fatal(err)
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
	if err != nil { // chaange to err == right err
		// if no char, check if account exists in DB, create char
		stmt, err = db.Prepare("SELECT account_name FROM accounts WHERE LOWER(account_name) = LOWER(?)")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		err = stmt.QueryRow(char).Scan(&acc)
		if err != nil { // change to err == right err
			//if no acct, create acccount
			tx, err := db.Begin()
			if err != nil {
				log.Fatal(err)
			}

			stmt, err := tx.Prepare("INSERT INTO accounts (account_name) VALUES(?)")
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()

			_, err = stmt.Exec(acct)
			if err != nil {
				log.Fatal(err)
			}
			tx.Commit()
		} else if err != nil {
			log.Fatal(err)
		}
		// create character
		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}

		stmt, err := tx.Prepare("INSERT INTO chars VALUES(%s, %s, %s, %s, %s, %s, 't')")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(acct, char, class, race, lvl, date)
		if err != nil {
			log.Fatal(err)
		}
		tx.Commit()
	} else if err != nil {
		log.Fatal(err)
	}
}
