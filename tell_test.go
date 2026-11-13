package main

import (
	"fmt"
	"strings"
	"testing"
)

// Test_CapTells checks that no reply sends the MUD more than maxReplyBytes,
// since the MUD disconnects a character that sends too much at once, and
// that a reply cut short keeps as many tells as fit and says how many more
// there were.
func Test_CapTells(t *testing.T) {
	size := func(tells []string) int {
		n := 0
		for _, tell := range tells {
			n += len(tell) + 1 // tintin ends each with \r\n
		}
		return n
	}
	// n tells of 40 to 100 bytes, about a clist line's length on average
	tells := func(n int) []string {
		var txt []string
		for i := 1; i <= n; i++ {
			txt = append(txt, fmt.Sprintf("t Yog %d %s\n",
				i, strings.Repeat("x", 30+i*7%60)))
		}
		return txt
	}

	// a reply that fits goes out whole
	fits := 0
	for size(tells(fits+1)) <= maxReplyBytes {
		fits++
	}
	chk(t, "capTells() of a reply that just fits", tells(fits),
		capTells("Yog", tells(fits)))

	// one that doesn't keeps as many tells as fit with the count of the
	// rest: one tell too many, clist aizu, and clist kalthazz
	for _, n := range []int{fits + 1, 94, 523} {
		in := tells(n)
		out := capTells("Yog", in)
		k := len(out) - 1
		if k < 1 {
			t.Errorf("capTells() of %d tells check failed: %q", n, out)
			continue
		}
		chk(t, fmt.Sprintf("capTells() of %d tells", n),
			append(in[:k:k], moreTells("Yog", n-k)), out)
		if size(out) > maxReplyBytes {
			t.Errorf("capTells() of %d tells check failed: %d bytes",
				n, size(out))
		}
		// one more, with the count of the rest after it, wouldn't fit
		if size(in[:k+1])+len(moreTells("Yog", n-k-1))+1 <= maxReplyBytes {
			t.Errorf("capTells() of %d tells check failed: only %d kept",
				n, k)
		}
	}

	chk(t, "moreTells() of one",
		[]string{"t Yog ...and 1 more tell, too many to send at once.\n"},
		[]string{moreTells("Yog", 1)})
}

// Test_TitleCase checks the capitals addname gives a real name: one at the
// start of each word, with the rest left as typed.
func Test_TitleCase(t *testing.T) {
	for in, good := range map[string]string{
		"":                  "",
		"Bob":               "Bob",
		"john smith":        "John Smith",
		"mary-jane o'brien": "Mary-Jane O'Brien",
		"JOHN mcDonald":     "JOHN McDonald",
		"  two  spaces":     "  Two  Spaces",
		"jean_luc 2nd":      "Jean_luc 2nd",
		"élodie":            "Élodie",
	} {
		if got := titleCase(in); got != good {
			t.Errorf("titleCase(%q) check failed: %q, should be %q",
				in, got, good)
		}
	}
}
