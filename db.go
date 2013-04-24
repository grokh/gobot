package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func OpenDB() *sql.DB {
	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		log.Fatalf("Fatal Error: Cannot open DB: %v", err)
	}
	return db
}

func ChkRows(rows *sql.Rows) {
	err := rows.Err()
	if err != nil {
		log.Fatalf("Fatal Error: Rows returned error: %v", err)
	}
	rows.Close()
}
