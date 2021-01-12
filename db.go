package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os/exec"
)

func OpenDB() *sql.DB {
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
	cmd := exec.Command("sh", "-c",
		"echo '.dump' | sqlite3 toril.db | "+
			"gzip -c >bak/toril.db.`date +\"%Y-%m-%d\"`.gz")
	err := cmd.Run()
	if err != nil {
		log.Fatalln("Fatal Error: Cannot backup DB: ", err)
	}
	// restore: cat dumpfile.sql | sqlite3 my_database.sqlite
}

func RestoreDB(file string) { // this doesn't work on Mac OS X
	cmd := exec.Command("sh", "-c", "zcat "+file+" | sqlite3 toril.db")
	err := cmd.Run()
	if err != nil {
		log.Fatalln("Fatal Error: Cannot restore DB: ", err)
	}
}
