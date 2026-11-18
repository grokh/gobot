package main

import "time"

// RecordOnline adds one who's count of players online, from katumi.tin's
// "Total players online" trigger, to the hour it was made in, in toril.db's
// online table (see init_db.sql): the who checks made that hour, their
// total, and the fewest and most at once.
func RecordOnline(n int) {
	db := OpenDB()
	defer db.Close()
	_, err := db.Exec("INSERT INTO online (hour, checks, total, low, high) "+
		"VALUES (?, 1, ?, ?, ?) ON CONFLICT (hour) DO UPDATE SET "+
		"checks = checks + 1, total = total + excluded.total, "+
		"low = min(low, excluded.low), high = max(high, excluded.high)",
		onlineHour(now()), n, n, n)
	ChkErr(err)
}

// onlineHour is the online table's key for the hour t is in: the hour's
// start in UTC, written as gobot writes its other times.
func onlineHour(t time.Time) string {
	return t.UTC().Truncate(time.Hour).Format("2006-01-02 15:04:05-07:00")
}
