package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
	"time"
)

func Uptime(curup string) {
	split := strings.Split(curup, ":")
	curboot := split[0] + "h" + split[1] + "m" + split[2] + "s"
	curtime, err := time.ParseDuration(curboot)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("SELECT boot_id, uptime FROM boots WHERE boot_time = (SELECT MAX(boot_time) FROM boots)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var oldid int
	var oldup string
	err = stmt.QueryRow().Scan(&oldid, &oldup)
	if err != nil {
		log.Fatal(err)
	}
	split = strings.Split(oldup, ":")
	oldboot := split[0] + "h" + split[1] + "m" + split[2] + "s"
	oldtime, err := time.ParseDuration(oldboot)
	if err != nil {
		log.Fatal(err)
	}

	if curtime < oldtime {
		// it's a new boot, so create a new boot ID and send email
		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}

		boottime := time.Now().Add(-curtime)
		stmt, err = db.Prepare("INSERT INTO boots (boot_time, uptime) VALUES(?, ?)")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(boottime, curup)
		if err != nil {
			log.Fatal(err)
		}
		tx.Commit()
		// SendBootEmail needs to be created in tokens.go
		// It should define and send the three variables
		// required by/to email.Notify(from, to, pwd)
		SendBootEmail()
	} else {
		// it's still the current boot, so update current uptime
		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}
		stmt, err := tx.Prepare("UPDATE boots SET uptime = ? WHERE boot_id = ?")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(curup, oldid)
		if err != nil {
			log.Fatal(err)
		}
		tx.Commit()
	}
}
