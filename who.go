package main

import (
    _ "github.com/bmizerany/pq"
    "database/sql"
    "fmt"
)

func Who(name string) {
	db, err := sql.Open("postgres", "user=kalkinine dbname=torildb sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sql := "SELECT account_name, char_name FROM chars WHERE LOWER(char_name) = LOWER($1)"
	q, err := db.Prepare(sql)
	defer q.Close()

	var acc string
	var char string
	err = q.QueryRow(name).Scan(&acc, &char)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Char: %s, Acct: %s\n", acc, char)

	/* HowTo: select lots of rows and iterate over all them
	q := "SELECT account_name, char_name FROM chars WHERE LOWER(char_name) = LOWER($1)"
	rows, err := db.Query(q, name)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var acc string
		var char string
		err = rows.Scan(&acc, &char)
		fmt.Printf("Character %s exists in the database under the account %s.\n", char, acc)
	}
	err = rows.Err() // get any error encountered during iteration
	if err != nil {
		panic(err)
	}
	*/
}

