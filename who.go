package main

import (
	_ "github.com/mattn/go-sqlite3"
	"database/sql"
	"fmt"
)

func Who(name string) {
	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("SELECT account_name, char_name FROM chars WHERE LOWER(char_name) = LOWER(?)")
        if err != nil {
                fmt.Println(err)
                return
        }
        defer stmt.Close()
        var acc string
	var char string
        err = stmt.QueryRow(name).Scan(&acc, &char)
        if err != nil {
                fmt.Println(err)
                return
        }
	fmt.Printf("Char: %s, Acct: %s\n", acc, char)
}
