package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func OpenDB() *sql.DB {
	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		log.Fatal("Fatal Error: Cannot open DB: " + err)
	}
	return db
}

func ChkRows(rows *sql.Rows) {
	err := rows.Err()
	if err != nil {
		log.Fatal("Fatal Error: Rows returned error: " + err)
	}
	rows.Close()
}
