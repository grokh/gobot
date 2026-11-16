package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
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

	// begun IMMEDIATE (see OpenDB), so it can read before it writes
	tx, err := db.Begin()
	ChkErr(err)
	// a fatal error below saves nothing and frees the lock; after Commit,
	// this does nothing
	defer tx.Rollback()
	query := "UPDATE chars SET char_level = ?, last_seen = ? " +
		"WHERE char_name = ?"
	stmt, err := tx.Prepare(query)
	ChkErr(err)
	defer stmt.Close()

	date := now().UTC()
	var name string
	for _, who := range ppl {
		char := re.FindStringSubmatch(who)
		//log.Println(char)
		if len(char) == 5 {
			lvl, _ := strconv.Atoi(char[1]) // one or two digits
			name = char[3]
			// the level the DB had, to tell if it fell
			var oldLvl int
			err := tx.QueryRow("SELECT char_level FROM chars "+
				"WHERE char_name = ?", name).Scan(&oldLvl)
			if err != nil && err != sql.ErrNoRows {
				fatal(err)
			}
			res, err := stmt.Exec(lvl, date, name)
			if err != nil {
				fatal(err)
			} else {
				affected, err := res.RowsAffected()
				if err != nil {
					fatal(err)
				} else if affected != 1 {
					// not in the DB yet: an individual who adds it
					cmd := fmt.Sprintf("who %s\n", name)
					cmds = append(cmds, cmd)
				} else {
					// all three, as each records or logs what it finds
					class := newClass(tx, name, char[2])
					race := newBatchRace(tx, name, char[4])
					fell := levelFell(name, oldLvl, lvl)
					if class || race || fell {
						// look it up again, so c.who() can catch a new
						// account
						cmd := fmt.Sprintf("who %s\n", name)
						cmds = append(cmds, cmd)
					}
				}
			}
		}
	}

	ChkErr(tx.Commit())
	return cmds
}

// newBatchRace catches a char the who list shows with a different race
// than the DB has, as when a char is deleted and someone makes another of
// the same name, and records the race shown, as the races table spells it.
// As with newClass, the race comes from the who list itself, so the two then
// agree and the char isn't flagged again. Text in the race's place that
// isn't a race the table has, like a flag after it, is left alone: a who
// line comes every 30 seconds, and it would send a lookup each time.
func newBatchRace(tx *sql.Tx, name string, shown string) bool {
	query := "SELECT c.char_race, r.race_name FROM chars c, races r " +
		"WHERE c.char_name = ? AND LOWER(r.race_name) = LOWER(?)"
	var stored, race string
	err := tx.QueryRow(query, name, shown).Scan(&stored, &race)
	if err == sql.ErrNoRows {
		return false
	}
	ChkErr(err)
	if strings.EqualFold(stored, race) {
		return false
	}
	_, err = tx.Exec("UPDATE chars SET char_race = ? WHERE char_name = ?",
		race, name)
	ChkErr(err)
	log.Printf("Race change: %s from %s to %s\n", name, stored, race)
	return true
}

// levelFell reports whether a char's level is lower than the DB had it, and
// logs it if so. Nearly always that's a char started over at level 1:
// deleted and made again under its name, by its player or by someone else.
// A char restored from a backup can come back a few levels down, and a lich
// loses levels, though that comes with a new class anyway.
func levelFell(name string, old int, lvl int) bool {
	if lvl >= old {
		return false
	}
	log.Printf("Level drop: %s from %d to %d\n", name, old, lvl)
	return true
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

// acctName is what an account's name looks like: every one the MUD has shown
// is letters only.
var acctName = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// lightFlag is the one thing the MUD puts after a who line's account, with
// the ) before it: (Daylight) or (In Dark), for a char the light bothers.
var lightFlag = regexp.MustCompile(`(?i)\)\s*\((daylight|in dark)$`)

// whoAcct picks the account out of what katumi.tin's who triggers take for
// it: everything from the line's first "(@" to its last ")". That's the
// account alone on nearly every line, but a light flag can follow it, and a
// title comes before it, where a player could write a "(@name)" of their
// own. The MUD's account is the last one on the line, so that's the one
// read. ok is false if what's left isn't an account's name.
func whoAcct(s string) (acct string, ok bool) {
	s = strings.TrimSpace(s)
	for {
		loc := lightFlag.FindStringIndex(s)
		if loc == nil {
			break
		}
		s = s[:loc[0]]
	}
	if i := strings.LastIndex(s, "(@"); i >= 0 {
		s = s[i+len("(@"):]
	}
	return s, acctName.MatchString(s)
}

func (c *Char) who() []string {
	var txt []string
	c.tseen = now().UTC()
	c.race = strings.TrimSpace(c.race)
	// Since a char moves to the account its who line shows, a line read
	// wrong mustn't make an account or move anyone to it.
	shown, ok := whoAcct(c.acct)
	if !ok {
		log.Printf("Who line skipped: %s's account read as %q\n",
			c.name, c.acct)
		return txt
	}
	c.acct = shown

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
		ChkErr(tx.Commit())
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
