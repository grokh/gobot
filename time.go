package main

import (
	"database/sql"
	"strings"
	"time"
)

// sendBootAlert is what Uptime calls on a new boot; the tests swap it out.
var sendBootAlert = SendBootAlert

// bootSlack is how much later than the old boot was last seen running the
// MUD has to have come up before it counts as a new boot, so a second or
// two of clock difference between the MUD and this server can't trip it.
const bootSlack = time.Minute

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

	// It's a new boot if the uptime went down, or if the MUD came up after
	// the old boot was last seen running. The second case catches a reboot
	// while the bot was offline for longer than the old boot had lasted,
	// when the new uptime can already be past the old one.
	boottime := time.Now().UTC().Add(-curtime)
	lastSeen := time.Unix(started.Int64, 0).Add(oldtime)
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
