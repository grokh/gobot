package main

import (
	"database/sql"
	// _ "github.com/bmizerany/pq"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func OpenDB() *sql.DB {
	// db, err := sql.Open("postgres", "user=kalkinine dbname=torildb sslmode=disable")
	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		log.Fatalln("Fatal Error: Cannot open DB: ", err)
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
