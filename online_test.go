package main

import (
	"testing"
	"time"
)

func Test_OnlineHour(t *testing.T) {
	for _, c := range []struct {
		t    time.Time
		good string
	}{
		{time.Date(2026, 11, 5, 15, 49, 18, 0, time.FixedZone("EST", -5*3600)),
			"2026-11-05 20:00:00+00:00"},
		{time.Date(2026, 11, 5, 23, 59, 59, 999, time.UTC),
			"2026-11-05 23:00:00+00:00"},
	} {
		if hour := onlineHour(c.t); hour != c.good {
			t.Errorf("onlineHour(%s) check failed: %s, not %s", c.t, hour, c.good)
		}
	}
}

// testOnline checks that who's counts add up by hour, in a toril.db made
// before the online table too.
func testOnline(t *testing.T) {
	db := OpenDB()
	defer db.Close()
	_, err := db.Exec("DROP TABLE online")
	chkErr(t, err)
	for _, n := range []int{17, 21, 15} {
		RecordOnline(n)
	}
	var hour time.Time
	var checks, total, low, high int
	chkErr(t, db.QueryRow("SELECT hour, checks, total, low, high FROM online").
		Scan(&hour, &checks, &total, &low, &high))
	if want := now().UTC().Truncate(time.Hour); !hour.Equal(want) ||
		checks != 3 || total != 53 || low != 15 || high != 21 {
		t.Errorf("RecordOnline() check failed: %s %d %d %d %d, not %s 3 53 15 21",
			hour, checks, total, low, high, want)
	}
}
