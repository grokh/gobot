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

// startSlack is how far apart the starts that two checks of one boot give
// can be: the uptime comes in whole seconds, and the line takes a moment to
// reach tintin. It's more than that needs, and it matters only in a boot's
// first seconds, when half its uptime is less.
const startSlack = 10 * time.Second

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

// seenAt is when tintin saw a time check: the Unix seconds katumi.tin
// passes as -seen, or gobot's own clock if there are none.
func seenAt(unix int64) time.Time {
	if unix == 0 {
		return now()
	}
	return time.Unix(unix, 0)
}

// Uptime records the uptime of a time check that tintin saw at seen: as the
// current boot's, or as a new boot's first, with an alert.
//
// seen less the uptime is when the MUD came up, to within a second or two,
// however long the check waits to be written. tintin runs each in the
// background, and when the MUD lags it answers several at once, so they can
// be written in any order. A check is the stored boot's if the start it
// gives is nearer that boot's start than where the boot was last seen, half
// its uptime later, and a new boot's if it's later than that: a reboot the
// bot is around for, one while it was away, and each boot of a crash loop.
// Half the uptime, not a fixed slack, as the MUD's uptime gains or loses up
// to about 6 seconds a day on this server's clock, which moves the start a
// long boot's checks give, while some reboots are back within seconds of
// Katumi's last check.
func Uptime(curup string, seen time.Time) {
	curtime := parseUptime(curup)
	start := seen.UTC().Add(-curtime)

	db := OpenDB()
	defer db.Close()

	// the current boot, the same one lr reports on, with its start in
	// whole seconds (a start that can't be read is 1970, so the check is a
	// new boot)
	query := "SELECT boot_id, uptime, " +
		"CAST(STRFTIME('%s', boot_time) AS INTEGER) " +
		"FROM boots ORDER BY boot_id DESC LIMIT 1"
	var oldid int
	var oldup string
	var started sql.NullInt64
	err := db.QueryRow(query).Scan(&oldid, &oldup, &started)
	ChkErr(err)
	oldtime := parseUptime(oldup)
	oldstart := time.Unix(started.Int64, 0)

	switch {
	case start.Sub(oldstart) > max(oldtime/2, startSlack):
		// written before the alert goes, so an alert that fails can't lose
		// the boot
		_, err = db.Exec("INSERT INTO boots (boot_time, uptime) VALUES(?, ?)",
			start, curup)
		ChkErr(err)
		sendBootAlert()
	case seen.Before(oldstart.Add(-startSlack)):
		// an older boot's check, held up past the reboot: its uptime isn't
		// this boot's
		log.Printf("Old check: uptime %s seen before boot %d began\n",
			curup, oldid)
	case curtime < oldtime:
		// this boot's, written after a newer one, whose uptime stands
		log.Printf("Late check: uptime %s written after %s\n", curup, oldup)
	case start.Before(oldstart):
		// A newer check that gives an earlier start moves the boot's back to
		// it, which puts it right when the boot's first check was seen late,
		// and in a new toril.db, whose first boot is only a stand-in.
		_, err = db.Exec("UPDATE boots SET boot_time = ?, uptime = ? "+
			"WHERE boot_id = ?", start, curup, oldid)
		ChkErr(err)
	default:
		_, err = db.Exec("UPDATE boots SET uptime = ? WHERE boot_id = ?",
			curup, oldid)
		ChkErr(err)
	}
}
