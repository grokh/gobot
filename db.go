package main

import (
	"database/sql"
	//_ "github.com/bmizerany/pq"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os/exec"
	"strings"
)

func OpenDB() *sql.DB {
	// postgres: sql.Open("postgres", "user=kalkinine dbname=torildb sslmode=disable")
	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		log.Fatalln("Fatal Error: Cannot open DB: ", err)
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Fatalln("Fatal Error: Cannot enforce foreign keys: ", err)
	}
	return db
}

func ChkRows(rows *sql.Rows) {
	err := rows.Err()
	if err != nil {
		log.Fatalln("Fatal Error: Rows returned error: ", err)
	}
	rows.Close()
}

func BackupDB() {
	// postgres: cmd := exec.Command("sh", "-c",
	// 	"pg_dump -U kalkinine torildb | "+
	// 		gzip > bak/torildb.`date +\"%Y-%m-%d\"`.sql.gz")
	cmd := exec.Command("sh", "-c",
		"echo '.dump' | sqlite3 toril.db | "+
			"gzip -c >bak/toril.db.`date +\"%Y-%m-%d\"`.gz")
	err := cmd.Run()
	if err != nil {
		log.Fatalln("Fatal Error: Cannot backup DB: ", err)
	}
}

func RestoreDB(file string) { // this doesn't work on Mac OS X
	// postgres: cmd := exec.Command("sh", "-c",
	//	"gunzip -c "+file+" | psql -U kalkinine -d torildb")
	cmd := exec.Command("sh", "-c", "zcat "+file+" | sqlite3 toril.db")
	err := cmd.Run()
	if err != nil {
		log.Fatalln("Fatal Error: Cannot restore DB: ", err)
	}
}

func Weather(oper string) []string {
	out, err := exec.Command("wu", "-conditions", "-s=\""+oper+"\"").Output()
	ChkErr(err)
	weather := strings.Split(strings.TrimSpace(string(out)), "\n")
	return weather
}
