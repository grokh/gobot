package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"os/exec"
)

func OpenDB() *sql.DB {
	db, err := sql.Open("sqlite3", "toril.db")
	if err != nil {
		fatal("Fatal Error: Cannot open DB: ", err)
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		fatal("Fatal Error: Cannot enforce foreign keys: ", err)
	}
	return db
}

// torilDBFile is TorilDB.db, the stats DB built on top of toril.db and
// published separately, kept next to toril.db and refreshed daily.
// FindItem falls back to its eq table when toril.db has no match.
const torilDBFile = "TorilDB.db"

// OpenTorilDB opens TorilDB.db read-only. It returns nil when the file
// isn't there, instead of creating an empty one as a plain sql.Open
// would, so lookups just answer from toril.db alone.
func OpenTorilDB() *sql.DB {
	if _, err := os.Stat(torilDBFile); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("TorilDB.db: %v\n", err)
		}
		return nil
	}
	db, err := sql.Open("sqlite3", "file:"+torilDBFile+"?mode=ro")
	if err != nil {
		log.Printf("TorilDB.db: cannot open: %v\n", err)
		return nil
	}
	return db
}

func ChkRows(rows *sql.Rows) {
	err := rows.Err()
	if err != nil {
		fatal("Fatal Error: Rows returned error: ", err)
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
