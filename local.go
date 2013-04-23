package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
)

func GlistStats(list string) {
	db, err := sql.Open("sqlite3", "toril.db")
	ChkErr(err)
	defer db.Close()

	// query items table for exact item name
	list = strings.Trim(list, "| ")
	//log.Printf("List: %v\n", list) // debug
	items := strings.Split(list, "|")
	//log.Printf("Items: %v\n", items) // debug
	query := "SELECT short_stats FROM items WHERE item_name = ?"

	var stat string
	for _, item := range items {
		//log.Printf("Item: %s\n", item) // debug
		item = item[32:]
		//log.Printf("Trimmed: %s\n", item) // debug
		stmt, err := db.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		err = stmt.QueryRow(item).Scan(&stat)
		log.Println(stat)
		if err == sql.ErrNoRows {
			fmt.Printf("%s is not in the database.\n", item)
		} else if err != nil {
			log.Fatal(err)
		} else {
			fmt.Printf("%s\n", stat)
		}
	}
}
