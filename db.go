package main

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// recordsDown is the reply to a tell that needs toril.db when it can't be
// read or written: locked by a long read, say, or broken. Nothing the tell
// would have changed is saved.
const recordsDown = "Katumi's records are unavailable right now. " +
	"Please try again later."

// dbWait is how long gobot waits on toril.db while another process holds
// it, as a long read does, before giving up with "database is locked". It's
// go-sqlite3's own default.
var dbWait = 5 * time.Second

// OpenDB opens toril.db. Its transactions begin with BEGIN IMMEDIATE,
// which takes the write lock before anything else, waiting up to dbWait
// for it as a single write does. SQLite's default transaction takes the
// lock at its first write, and one that has read by then gets "database is
// locked" at once if another process is writing: SQLite doesn't wait there,
// as that process may be waiting for the read to end. WhoBatch, which reads
// each char's level before updating it, failed that way now and then.
func OpenDB() *sql.DB {
	db, err := sql.Open("sqlite3", fmt.Sprintf(
		"toril.db?_busy_timeout=%d&_txlock=immediate", dbWait.Milliseconds()))
	if err != nil {
		fatal("Fatal Error: Cannot open DB: ", err)
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		// closed, since a tell carries on past fatal to answer (and
		// Windows won't delete a test's toril.db while it's open)
		db.Close()
		fatal("Fatal Error: Cannot enforce foreign keys: ", err)
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

// dumpWait is how long a backup waits for a write by the bot to finish.
var dumpWait = 10 * time.Second

// keepDays is how many days of daily backups -bak keeps. Older dailies are
// deleted, except the first backup of each month, which is kept for good.
const keepDays = 60

// botTables are the tables init_db.sql makes. The bot's toril.db has all of
// them, and more (the old item tables), so -bak and -res refuse a database
// that lacks one, as it isn't the bot's: an empty one, which is what the
// sqlite3 CLI makes of a toril.db that isn't there, or the copy dbcopy.sh
// publishes (in the gobot checkout, among other places), which has no
// online. Test_BotTables keeps the list in step with init_db.sql.
var botTables = []string{"accounts", "boots", "chars", "class_types",
	"classes", "loads", "online", "race_types", "races"}

// openReadOnly opens the SQLite database at path read-only, so it's never
// made or changed: one that's missing fails at the first query. A write by
// the bot is waited for, up to dumpWait.
func openReadOnly(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro&_busy_timeout=%d",
		filepath.ToSlash(path), dumpWait.Milliseconds()))
}

// tables lists the tables in the database at path, opened read-only.
func tables(path string) (map[string]bool, error) {
	db, err := openReadOnly(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query("SELECT name FROM sqlite_schema WHERE type = 'table'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	have := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		have[name] = true
	}
	return have, rows.Err()
}

// checkTables checks that the database at path has every table in
// botTables. name is what its errors call the database.
func checkTables(path, name string) error {
	have, err := tables(path)
	if err != nil {
		return fmt.Errorf("%s: %v", name, err)
	}
	var missing []string
	for _, table := range botTables {
		if !have[table] {
			missing = append(missing, table)
		}
	}
	switch {
	case len(have) == 0:
		return fmt.Errorf("%s has no tables", name)
	case len(missing) == 1:
		return fmt.Errorf("%s has no %s table, so it isn't the bot's database",
			name, missing[0])
	case len(missing) > 1:
		return fmt.Errorf("%s has no %s tables, so it isn't the bot's database",
			name, strings.Join(missing, ", "))
	}
	return nil
}

// BackupDB saves a gzipped .dump of toril.db in bak/, the form RestoreDB
// reads back, then prunes old dailies. toril.db has to be there, with every
// table in botTables, and is only read, so a missing one fails instead of
// being made empty and saved as the day's backup. The dump goes to a temp
// file first and is only renamed into place once it's known to be
// complete, so a failed backup leaves the day's earlier one as it was.
func BackupDB() error {
	if err := checkTables("toril.db", "toril.db"); err != nil {
		return err
	}
	if err := os.MkdirAll("bak", 0750); err != nil {
		return err
	}
	name := filepath.Join("bak",
		"toril.db."+time.Now().Format("2006-01-02")+".gz")
	tmp := name + ".tmp"
	if err := writeDump(tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, name); err != nil {
		return err
	}
	pruneBackups(time.Now())
	return nil
}

// writeDump gzips a .dump of toril.db, opened read-only, into path. sqlite3
// exits 0 even when a dump stops partway, as it does on a locked database,
// so this also checks that it printed no errors and that the dump reached
// its COMMIT.
func writeDump(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := gzip.NewWriter(f)
	var end lastBytes
	var stderr bytes.Buffer
	cmd := exec.Command("sqlite3", "-readonly", "-cmd",
		fmt.Sprintf(".timeout %d", dumpWait.Milliseconds()),
		"toril.db", ".dump")
	cmd.Stdout = io.MultiWriter(zw, &end)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	if stderr.Len() > 0 {
		return errors.New(strings.TrimSpace(stderr.String()))
	}
	if !strings.HasSuffix(strings.TrimSpace(string(end.b)), "COMMIT;") {
		return errors.New("dump didn't finish")
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return f.Close()
}

// lastBytes keeps the last few bytes written to it.
type lastBytes struct{ b []byte }

func (l *lastBytes) Write(p []byte) (int, error) {
	l.b = append(l.b, p...)
	if len(l.b) > 64 {
		l.b = append([]byte(nil), l.b[len(l.b)-64:]...)
	}
	return len(p), nil
}

// pruneBackups deletes daily backups more than keepDays old, except the
// first one of each month, which stays however old it gets. Only files
// named the way BackupDB names them are touched.
func pruneBackups(now time.Time) {
	names, err := filepath.Glob(filepath.Join("bak", "toril.db.????-??-??.gz"))
	ChkErr(err)
	sort.Strings(names) // by date, oldest first
	cutoff := now.AddDate(0, 0, -keepDays).Format("2006-01-02")
	months := make(map[string]bool)
	removed := 0
	for _, name := range names {
		day := strings.TrimSuffix(
			strings.TrimPrefix(filepath.Base(name), "toril.db."), ".gz")
		if _, err := time.Parse("2006-01-02", day); err != nil {
			continue
		}
		if !months[day[:7]] {
			months[day[:7]] = true // the month's first backup
			continue
		}
		if day < cutoff {
			ChkErr(os.Remove(name))
			removed++
		}
	}
	if removed > 0 {
		log.Printf("Backups: deleted %d dailies more than %d days old, "+
			"kept the first of each month\n", removed, keepDays)
	}
}

// RestoreDB makes toril.db from file, a backup -bak wrote, and says what it
// holds. The dump is loaded into a new database beside toril.db, which takes
// its place only once the file has decompressed whole, the dump has ended in
// COMMIT and loaded without an error, and the result passes SQLite's
// integrity check and has every table in botTables. A toril.db with tables
// is never replaced: move it aside first. A missing or empty one is, since
// gobot leaves an empty one when it runs while toril.db is away.
func RestoreDB(file string) (string, error) {
	if err := canReplace(); err != nil {
		return "", err
	}
	dump, err := gunzip(file)
	if err != nil {
		return "", err
	}
	if !bytes.HasSuffix(bytes.TrimSpace(dump), []byte("COMMIT;")) {
		return "", errors.New("the dump doesn't end in COMMIT: it stopped partway")
	}

	// made 0644 less the umask, as SQLite makes a database, rather than
	// os.CreateTemp's 0600
	tmp := fmt.Sprintf("toril.db.restore-%d", os.Getpid())
	f, err := os.OpenFile(tmp, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return "", err
	}
	f.Close()
	installed := false
	defer func() {
		if !installed {
			os.Remove(tmp)
		}
		os.Remove(tmp + "-journal")
	}()
	// -bail, so a dump that won't load stops at its first error, rather
	// than going on to commit what was left
	var stderr bytes.Buffer
	cmd := exec.Command("sqlite3", "-bail", tmp)
	cmd.Stdin = bytes.NewReader(dump)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("loading the dump: %v: %s",
			err, strings.TrimSpace(stderr.String()))
	}
	if stderr.Len() > 0 {
		return "", fmt.Errorf("loading the dump: %s",
			strings.TrimSpace(stderr.String()))
	}

	if err := checkTables(tmp, "the backup"); err != nil {
		return "", err
	}
	db, err := openReadOnly(tmp)
	if err != nil {
		return "", err
	}
	var check string
	var chars, accounts, boots int
	err = db.QueryRow("PRAGMA integrity_check").Scan(&check)
	if err == nil {
		err = db.QueryRow("SELECT (SELECT count(*) FROM chars), "+
			"(SELECT count(*) FROM accounts), (SELECT count(*) FROM boots)").
			Scan(&chars, &accounts, &boots)
	}
	db.Close() // before the rename: Windows won't rename an open file
	if err != nil {
		return "", err
	}
	if check != "ok" {
		return "", fmt.Errorf("the backup fails SQLite's integrity check: %s",
			check)
	}

	// again, in case a toril.db came back while the dump loaded
	if err := canReplace(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, "toril.db"); err != nil {
		return "", err
	}
	installed = true
	return fmt.Sprintf("Restored toril.db from %s: %d chars, %d accounts, "+
		"%d boots.", file, chars, accounts, boots), nil
}

// canReplace checks that a restore may put a new toril.db in place: there's
// no toril.db, or only one with no tables, and no journal beside it. SQLite
// would play a journal the old toril.db left back into the new one the
// first time it's opened, corrupting it.
func canReplace() error {
	fi, err := os.Stat("toril.db")
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return err
	case fi.Size() > 0:
		have, err := tables("toril.db")
		if err != nil {
			return fmt.Errorf("toril.db can't be read (%v): move it aside first",
				err)
		}
		if len(have) > 0 {
			return errors.New("toril.db has tables: move it aside first")
		}
	}
	for _, journal := range []string{"toril.db-journal", "toril.db-wal"} {
		if _, err := os.Stat(journal); err == nil {
			return fmt.Errorf("%s is left from the old toril.db: "+
				"move it aside too", journal)
		}
	}
	return nil
}

// gunzip reads all of the gzip file at path, and fails if it isn't gzip, is
// cut short, or fails its checksum. zcat piped into sqlite3 would hide all
// three, since a pipe's exit status is its last command's.
func gunzip(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("can't decompress it: %v", err)
	}
	dump, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("can't decompress it: %v", err)
	}
	return dump, nil
}
