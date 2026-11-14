package main

import (
	"database/sql"
	"log"
	"strings"
	"time"
)

// sendBootAlert is what Uptime calls on a new boot; the tests swap it out.
var sendBootAlert = SendBootAlert

// now is the clock gobot stamps boots, sightings and load reports with.
// Test_All stops it, so the times it expects can't tick over mid-test.
var now = time.Now

// bootSlack is how much later than the old boot was last seen running the
// MUD has to have come up before it counts as a new boot, so a second or
// two of clock difference between the MUD and this server can't trip it.
const bootSlack = time.Minute

// maxLate is the longest a check is taken to have been held up between the
// MUD answering it and gobot writing it down. katumi.tin runs each check in
// the background, and twice in 2014 its checks stalled for four minutes,
// came through together, and one was written after a newer one. Ten minutes
// leaves room for a longer stall than that.
const maxLate = 10 * time.Minute

// parseUptime turns the MUD's H:M:S uptime into a duration.
func parseUptime(up string) time.Duration {
	split := strings.Split(up, ":")
	if len(split) != 3 {
		fatal("Fatal Error: uptime not in H:M:S form: ", up)
	}
	d, err := time.ParseDuration(split[0] + "h" + split[1] + "m" +
		split[2] + "s")
	ChkErr(err)
	return d
}

// Uptime records the uptime the MUD gave katumi.tin's time check: as the
// current boot's latest, or as a new boot's first, with an alert.
func Uptime(curup string) {
	curtime := parseUptime(curup)

	db := OpenDB()
	defer db.Close()

	// the current boot, the same one lr reports on
	query := "SELECT boot_id, uptime, " +
		"CAST(STRFTIME('%s', boot_time) AS INTEGER) " +
		"FROM boots ORDER BY boot_id DESC LIMIT 1"
	var oldid int
	var oldup string
	var started sql.NullInt64
	err := db.QueryRow(query).Scan(&oldid, &oldup, &started)
	ChkErr(err)
	oldtime := parseUptime(oldup)

	// when the MUD came up, going by this check, and when the stored boot
	// was last seen running
	boottime := now().UTC().Add(-curtime)
	lastSeen := time.Unix(started.Int64, 0).Add(oldtime)

	// A lower uptime isn't a reboot when the check was only held up and
	// written after a newer one. A late check is at most maxLate behind
	// that one, and the boot time it gives is the stored boot's own, later
	// by the hold-up: long before the boot was last seen. A reboot's is
	// after the boot was last seen, or, when the boot's own first check was
	// held up and its start stored that much late, up to maxLate before. In
	// a boot's first minutes there is no room between the two, so a lower
	// uptime there is always a reboot, as it has to be in a crash loop.
	late := started.Valid && curtime < oldtime &&
		oldtime-curtime <= maxLate &&
		boottime.Before(lastSeen.Add(-maxLate))
	if late {
		// the newer check's uptime stands, and so does its boot
		log.Printf("Late check: uptime %s written after %s\n", curup, oldup)
		return
	}

	// It's a new boot if the uptime went down, or if the MUD came up after
	// the old boot was last seen running. The second case catches a reboot
	// while the bot was offline for longer than the old boot had lasted,
	// when the new uptime can already be past the old one.
	newBoot := curtime < oldtime ||
		(started.Valid && boottime.After(lastSeen.Add(bootSlack)))

	tx, err := db.Begin()
	ChkErr(err)
	if newBoot {
		// it's a new boot, so create a new boot ID and send an alert
		_, err = tx.Exec(
			"INSERT INTO boots (boot_time, uptime) VALUES(?, ?)",
			boottime, curup)
		ChkErr(err)
		ChkErr(tx.Commit())
		// committed first, so an alert that fails can't lose the boot
		sendBootAlert()
	} else {
		// it's still the current boot, so update current uptime
		_, err = tx.Exec("UPDATE boots SET uptime = ? WHERE boot_id = ?",
			curup, oldid)
		ChkErr(err)
		ChkErr(tx.Commit())
	}
}
