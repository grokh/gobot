package main

import "testing"

func Test_Guesses(t *testing.T) {
	for _, c := range []struct {
		a, b string
		n    int
	}{
		{"stud", "stud", 0}, {"stdu", "stud", 1}, {"teh", "the", 1},
		{"stats", "stat", 1}, {"kriik", "kri'ik", 1}, {"", "abc", 3},
		{"kitten", "sitting", 3},
	} {
		if n := edits(c.a, c.b); n != c.n {
			t.Errorf("edits(%q, %q) check failed: %d, not %d", c.a, c.b, n,
				c.n)
		}
	}

	for cmd, good := range map[string]string{
		"stats": "stat", "STAT": "stat", "astats": "astat", "fsat": "fstat",
		"mstats": "mstat", "hlep": "help", "clsit": "clist",
		"addnmae": "addname", "delaltt": "delalt", "stat,": "stat",
		// too far from any command to be a typo of one
		"hello": "", "blah": "", "hi": "", "": "",
	} {
		if guess := guessCommand(cmd); guess != good {
			t.Errorf("guessCommand(%q) check failed: %q, not %q", cmd, guess,
				good)
		}
	}
}

// testGuesses checks the guesses tells get against the test TorilDB.db.
func testGuesses(t *testing.T) {
	checks := []struct {
		tell string
		good []string
	}{
		{"stat infernal stileto", []string{
			"t Yog 404 item not found: infernal stileto\n",
			"t Yog Did you mean the infernal stiletto of bane?\n"}},
		{"astat lordly plaet", []string{
			"t Yog 404 item not found: lordly plaet\n",
			"t Yog Did you mean a suit of lordly plate?\n"}},
		// the shortest first, as stat would find them
		{"stat mithril stdu", []string{
			"t Yog 404 item not found: mithril stdu\n",
			"t Yog Did you mean a tiny mithril stud or a tiny mithril stud " +
				"set with a ruby?\n"}},
		// mobs go by their names without Kegor's numbers
		{"mstat kriik", []string{
			"t Yog 404 mob not found: kriik\n",
			"t Yog Did you mean the pit fiend, Kri'ik?\n"}},
		{"mstat bartendr", []string{
			"t Yog 404 mob not found: bartendr\n",
			"t Yog Did you mean the bartender?\n"}},
		// a word near none in the names
		{"stat stiletto blah",
			[]string{"t Yog 404 item not found: stiletto blah\n"}},
		// words that are all in names, just not in the same one
		{"stat ruby longsword",
			[]string{"t Yog 404 item not found: ruby longsword\n"}},
		{"stats infernal stiletto", []string{"t Yog " + BadSyntax + "\n",
			"t Yog Did you mean 'tell katumi stat infernal stiletto'?\n"}},
		{"stat", []string{"t Yog " + BadSyntax + "\n",
			"t Yog See 'tell katumi help stat'.\n"}},
		{"help stats", []string{"t Yog 404 help file not found: stats\n",
			"t Yog Did you mean 'tell katumi help stat'?\n"}},
	}
	for _, c := range checks {
		chk(t, "ReplyTo(Yog, "+c.tell+")", c.good, ReplyTo("Yog", c.tell))
	}
	// -item, which Ynnd's ASSOC stat runs, doesn't guess
	chk(t, "FindItem(infernal stileto)",
		[]string{"404 item not found: infernal stileto"},
		FindItem("infernal stileto", "short_stats"))
}
