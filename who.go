package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

type Char struct {
	class, name, race, acct, seen string
	lvl                           int
	tseen                         time.Time
}

func WhoBatch(batch string) []string {
	var cmds []string
	batch = strings.Trim(batch, "| ")
	ppl := strings.Split(batch, "|")
	re, err := regexp.Compile(
		`^\[[ ]?(\d{1,2}) ([[:alpha:]-]{3})\] ([[:alpha:]]+) .*\((.*)\)`)
	ChkErr(err)

	db := OpenDB()
	defer db.Close()

	tx, err := db.Begin()
	ChkErr(err)
	query := "UPDATE chars SET char_level = ?, last_seen = ? " +
		"WHERE char_name = ?"
	stmt, err := tx.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	date := now().UTC()
	var lvl string
	var name string
	for _, who := range ppl {
		char := re.FindStringSubmatch(who)
		//log.Println(char)
		if len(char) == 5 {
			lvl = char[1]
			name = char[3]
			res, err := stmt.Exec(lvl, date, name)
			if err != nil {
				log.Fatal(err)
			} else {
				affected, err := res.RowsAffected()
				if err != nil {
					log.Fatal(err)
				} else if affected != 1 {
					// not in the DB yet: an individual who adds it
					cmd := fmt.Sprintf("who %s\n", name)
					cmds = append(cmds, cmd)
				} else if newClass(tx, name, char[2]) {
					// look it up again, so c.who() can catch a new race or
					// account
					cmd := fmt.Sprintf("who %s\n", name)
					cmds = append(cmds, cmd)
				}
			}
		}
	}

	tx.Commit()
	return cmds
}

// newClass catches a char the who list shows with a different class than
// the DB has, like a necro who has become a lich, and updates it to the
// class the abbreviation stands for. The class comes from the who list
// itself, so once it's updated the two agree and the char isn't flagged
// again. An abbreviation the classes table doesn't have is left alone.
func newClass(tx *sql.Tx, name string, abbr string) bool {
	query := "SELECT c.class_name, l.class_name FROM chars c, classes l " +
		"WHERE c.char_name = ? AND LOWER(l.class_abbr) = LOWER(?)"
	var stored, class string
	err := tx.QueryRow(query, name, abbr).Scan(&stored, &class)
	if err == sql.ErrNoRows {
		return false
	}
	ChkErr(err)
	if stored == class {
		return false
	}
	_, err = tx.Exec("UPDATE chars SET class_name = ? WHERE char_name = ?",
		class, name)
	ChkErr(err)
	log.Printf("Class change: %s from %s to %s\n", name, stored, class)
	return true
}

func (c *Char) who() []string {
	var txt []string
	c.tseen = now().UTC()
	c.race = strings.TrimSpace(c.race)

	db := OpenDB()
	defer db.Close()

	// check if character exists in DB
	query := "SELECT account_name, char_name, char_race FROM chars " +
		"WHERE LOWER(char_name) = LOWER(?)"
	stmt, err := db.Prepare(query)
	ChkErr(err)
	defer stmt.Close()
	var acct, race string
	err = stmt.QueryRow(c.name).Scan(&acct, &c.name, &race)
	if err == nil {
		// the account first, as newRace finds the char by it
		if strings.EqualFold(c.acct, acct) {
			c.acct = acct
		} else {
			txt = c.newAcct(db, acct)
		}
		if !strings.EqualFold(c.race, race) {
			c.newRace(db, race)
		}
	} else if err == sql.ErrNoRows {
		// if no char, check if account exists in DB, create char
		txt = c.account(db)
		// create character
		tx, err := db.Begin()
		ChkErr(err)

		c.class = strings.TrimSpace(c.class)
		query = "INSERT INTO chars VALUES(?, ?, ?, ?, ?, ?, 't')"
		stmt, err := tx.Prepare(query)
		ChkErr(err)
		defer stmt.Close()

		log.Printf(
			"New char: [%d %s] %s (%s) (@%s) seen %s",
			c.lvl, c.class, c.name, c.race, c.acct, c.tseen,
		)
		_, err = stmt.Exec(c.acct, c.name, c.class, c.race, c.lvl, c.tseen)
		ChkErr(err)
		tx.Commit()
	} else if err != nil {
		log.Fatal(err)
	}
	return txt
}

// account finds the who line's account in the DB, adding it if it's new, and
// leaves c.acct spelled the way the DB has it. A new account with a char of
// level 20 or under is a new player, so the welcome for them is returned.
func (c *Char) account(db *sql.DB) []string {
	var txt []string
	err := db.QueryRow("SELECT account_name FROM accounts "+
		"WHERE LOWER(account_name) = LOWER(?)", c.acct).Scan(&c.acct)
	if err == sql.ErrNoRows {
		log.Printf("New acct: @%s", c.acct)
		if c.lvl <= 20 {
			txt = append(txt,
				fmt.Sprintf(
					"nhc Welcome, %s. If you have any questions, "+
						"feel free to ask on this channel.",
					c.name,
				),
			)
		}
		_, err = db.Exec("INSERT INTO accounts (account_name) VALUES(?)",
			c.acct)
		ChkErr(err)
	} else if err != nil {
		log.Fatal(err)
	}
	return txt
}

// newAcct moves a char already in the DB to the account its who line shows.
// A name gets a new account when its char is deleted and someone else makes
// one of the same name, or when the char changes hands. Left under the old
// account, it would be listed among that player's alts, and looked up by
// their real name. Whether the char is hidden stays as it was: only its
// player unhides it.
func (c *Char) newAcct(db *sql.DB, old string) []string {
	txt := c.account(db)
	_, err := db.Exec("UPDATE chars SET account_name = ? "+
		"WHERE account_name = ? AND char_name = ?", c.acct, old, c.name)
	ChkErr(err)
	log.Printf("Account change: %s from @%s to @%s\n", c.name, old, c.acct)
	return txt
}

// newRace records the race the who line shows for a char already in the DB,
// spelled the way the races table has it. A race the table doesn't have is
// logged and left alone. The class isn't touched here: WhoBatch keeps it
// current from the who list.
func (c *Char) newRace(db *sql.DB, old string) {
	var race string
	err := db.QueryRow("SELECT race_name FROM races "+
		"WHERE LOWER(race_name) = LOWER(?)", c.race).Scan(&race)
	if err == sql.ErrNoRows {
		log.Printf("Race not updated, not in races table: %s (%s)\n",
			c.name, c.race)
		return
	}
	ChkErr(err)
	_, err = db.Exec("UPDATE chars SET char_race = ? "+
		"WHERE account_name = ? AND char_name = ?", race, c.acct, c.name)
	ChkErr(err)
	log.Printf("Race change: %s from %s to %s\n", c.name, old, race)
}
