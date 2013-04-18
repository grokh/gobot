package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"regexp"
	"strings"
	"time"
)

func WhoBatch(batch string) {
	batch = strings.Trim(batch, "| ")
	ppl := strings.Split(batch, "|")
	re, err := regexp.Compile(`^\[[ ]?(\d{1,2}) ([[:alpha:]-]{3})\] ([[:alpha:]]+) .*\((.*)\)`)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// TODO: also check class change for necro->lich
	// TODO: also check for account name change
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	query := "UPDATE chars SET char_level = ?, last_seen = ? " +
		"WHERE char_name = ?"
	stmt, err := tx.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}
	date := time.Now().In(loc)
	var lvl string
	var name string
	for _, who := range ppl {
		char := re.FindAllStringSubmatch(who, -1)
		//fmt.Println(char)
		if len(char) > 0 {
			if len(char[0]) == 5 {
				lvl = char[0][1]
				name = char[0][3]
				res, err := stmt.Exec(lvl, date, name)
				if err != nil {
					log.Fatal(err)
				} else {
					affected, err := res.RowsAffected()
					if err != nil {
						log.Fatal(err)
					} else {
						if affected != 1 {
							fmt.Printf("who %s\n", name)
						}
					}
				}
			}
		}
	}

	tx.Commit()
}

func WhoChar(char string, lvl int, class string, race string, acct string) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}
	date := time.Now().In(loc)
	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// check if character exists in DB
	query := "SELECT account_name, char_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?)"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var acc string
	var name string
	err = stmt.QueryRow(char).Scan(&acc, &name)
	if err == sql.ErrNoRows {
		// if no char, check if account exists in DB, create char
		query = "SELECT account_name FROM accounts " +
			"WHERE LOWER(account_name) = LOWER(?)"
		stmt, err = db.Prepare(query)
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		err = stmt.QueryRow(acct).Scan(&acc)
		if err == sql.ErrNoRows {
			//if no acct, create acccount
			tx, err := db.Begin()
			if err != nil {
				log.Fatal(err)
			}

			query = "INSERT INTO accounts (account_name) VALUES(?)"
			stmt, err := tx.Prepare(query)
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

		class = strings.Trim(class, " ")
		query = "INSERT INTO chars VALUES(?, ?, ?, ?, ?, ?, 't')"
		stmt, err := tx.Prepare(query)
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
