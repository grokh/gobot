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

// dumpWait is how long a backup waits for a write by the bot to finish.
var dumpWait = 10 * time.Second

// keepDays is how many days of daily backups -bak keeps. Older dailies are
// deleted, except the first backup of each month, which is kept for good.
const keepDays = 60

// BackupDB saves a gzipped .dump of toril.db in bak/, the form RestoreDB
// reads back, then prunes old dailies. The dump goes to a temp file first
// and is only renamed into place once it's known to be complete.
func BackupDB() {
	ChkErr(os.MkdirAll("bak", 0750))
	name := filepath.Join("bak",
		"toril.db."+time.Now().Format("2006-01-02")+".gz")
	tmp := name + ".tmp"
	if err := writeDump(tmp); err != nil {
		os.Remove(tmp)
		log.Fatalln("Fatal Error: Cannot backup DB: ", err)
	}
	ChkErr(os.Rename(tmp, name))
	pruneBackups(time.Now())
	// restore: zcat bak/toril.db.<date>.gz | sqlite3 toril.db
}

// writeDump gzips a .dump of toril.db into path. sqlite3 exits 0 even when
// a dump stops partway, as it does on a locked database, so this also
// checks that it printed no errors and that the dump reached its COMMIT.
func writeDump(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := gzip.NewWriter(f)
	var end lastBytes
	var stderr bytes.Buffer
	cmd := exec.Command("sqlite3", "-cmd",
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

func RestoreDB(file string) { // this doesn't work on Mac OS X
	cmd := exec.Command("sh", "-c", "zcat "+file+" | sqlite3 toril.db")
	err := cmd.Run()
	if err != nil {
		log.Fatalln("Fatal Error: Cannot restore DB: ", err)
	}
}
