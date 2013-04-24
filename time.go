package main

import (
	_ "github.com/mattn/go-sqlite3"
	"strings"
	"time"
)

func Uptime(curup string) {
	split := strings.Split(curup, ":")
	curboot := split[0] + "h" + split[1] + "m" + split[2] + "s"
	curtime, err := time.ParseDuration(curboot)
	ChkErr(err)
	loc, err := time.LoadLocation("America/New_York")
	ChkErr(err)

	db := OpenDB()
	defer db.Close()

	query := "SELECT boot_id, uptime FROM boots " +
		"WHERE boot_time = (SELECT MAX(boot_time) FROM boots)"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	var oldid int
	var oldup string
	err = stmt.QueryRow().Scan(&oldid, &oldup)
	ChkErr(err)
	split = strings.Split(oldup, ":")
	oldboot := split[0] + "h" + split[1] + "m" + split[2] + "s"
	oldtime, err := time.ParseDuration(oldboot)
	ChkErr(err)

	if curtime < oldtime {
		// it's a new boot, so create a new boot ID and send email
		tx, err := db.Begin()
		ChkErr(err)

		boottime := time.Now().In(loc).Add(-curtime)
		stmt, err = db.Prepare("INSERT INTO boots (boot_time, uptime) VALUES(?, ?)")
		ChkErr(err)
		defer stmt.Close()

		_, err = stmt.Exec(boottime, curup)
		ChkErr(err)
		tx.Commit()
		// make sure you have a tokens.txt file containing
		// gmail account on first line, pwd on second,
		// and each additional line containing the target emails
		SendBootEmail()
	} else {
		// it's still the current boot, so update current uptime
		tx, err := db.Begin()
		ChkErr(err)
		stmt, err := tx.Prepare("UPDATE boots SET uptime = ? WHERE boot_id = ?")
		ChkErr(err)
		defer stmt.Close()

		_, err = stmt.Exec(curup, oldid)
		ChkErr(err)
		tx.Commit()
	}
}
