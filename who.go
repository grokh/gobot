package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

func WhoBatch(batch string) []string {
	var cmds []string
	batch = strings.Trim(batch, "| ")
	ppl := strings.Split(batch, "|")
	re, err := regexp.Compile(
		`^\[[ ]?(\d{1,2}) ([[:alpha:]-]{3})\] ([[:alpha:]]+) .*\((.*)\)`)
	ChkErr(err)

	db := OpenDB()
	defer db.Close()

	// TODO: also check class change for necro->lich
	// TODO: also check for account name change
	tx, err := db.Begin()
	ChkErr(err)
	query := "UPDATE chars SET char_level = ?, last_seen = ? " +
		"WHERE char_name = ?"
	stmt, err := tx.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	loc, err := time.LoadLocation("America/New_York")
	ChkErr(err)
	date := time.Now().In(loc)
	var lvl string
	var name string
	for _, who := range ppl {
		char := re.FindStringSubmatch(who)
		//log.Println(char)
		if len(char) == 5 {
			lvl = char[1]
			name = char[3]
			res, err := stmt.Exec(lvl, date, name)
			if err != nil {
				log.Fatal(err)
			} else {
				affected, err := res.RowsAffected()
				if err != nil {
					log.Fatal(err)
				} else {
					if affected != 1 {
						cmd := fmt.Sprintf("who %s\n", name)
						cmds = append(cmds, cmd)
					}
				}
			}
		}
	}

	tx.Commit()
	return cmds
}

func WhoChar(
	char string,
	lvl int,
	class string,
	race string,
	acct string,
) []string {
	var txt []string
	loc, err := time.LoadLocation("America/New_York")
	ChkErr(err)
	date := time.Now().In(loc)

	db := OpenDB()
	defer db.Close()

	// check if character exists in DB
	query := "SELECT account_name, char_name FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?)"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()
	var acc string
	var name string
	err = stmt.QueryRow(char).Scan(&acc, &name)
	if err == sql.ErrNoRows {
		// if no char, check if account exists in DB, create char
		query = "SELECT account_name FROM accounts " +
			"WHERE LOWER(account_name) = LOWER(?)"
		stmt, err = db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		err = stmt.QueryRow(acct).Scan(&acc)
		if err == sql.ErrNoRows {
			//if no acct, create acccount
			tx, err := db.Begin()
			ChkErr(err)

			query = "INSERT INTO accounts (account_name) VALUES(?)"
			stmt, err := tx.Prepare(query)
			ChkErr(err)
			defer stmt.Close()

			log.Printf("New acct: @%s", acct)
			txt = append(txt,
				fmt.Sprintf(
					"nhc Welcome, %s. If you have any questions, "+
						"feel free to ask on this channel.",
					char,
				),
			)
			_, err = stmt.Exec(acct)
			ChkErr(err)
			tx.Commit()
		} else if err != nil {
			log.Fatal(err)
		}
		// create character
		tx, err := db.Begin()
		ChkErr(err)

		class = strings.TrimSpace(class)
		query = "INSERT INTO chars VALUES(?, ?, ?, ?, ?, ?, 't')"
		stmt, err := tx.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		log.Printf(
			"New char: [%d %s] %s (%s) (@%s) seen %s",
			lvl, class, char, race, acct, date,
		)
		_, err = stmt.Exec(acct, char, class, race, lvl, date)
		ChkErr(err)
		tx.Commit()
	} else if err != nil {
		log.Fatal(err)
	}
	return txt
}
