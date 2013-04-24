package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func OpenDB() *sql.DB {
	db, err := sql.Open("sqlite3", "toril.db")
	ChkErr(err)
	return db
}
