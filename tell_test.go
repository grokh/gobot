package main

import (
	"fmt"
	"testing"
)

// Test_CapTells checks that no reply goes past maxTells tells, since the MUD
// disconnects a character that sends too many commands at once, and that a
// reply cut short says how many more there were.
func Test_CapTells(t *testing.T) {
	tells := func(n int) []string {
		var txt []string
		for i := 1; i <= n; i++ {
			txt = append(txt, fmt.Sprintf("t Yog line %d\n", i))
		}
		return txt
	}
	chk(t, "capTells() at the limit", tells(maxTells),
		capTells("Yog", tells(maxTells)))

	// one over the limit, and clist aizu: 94 alts on the account
	for _, n := range []int{maxTells + 1, 94} {
		want := append(tells(maxTells-1), fmt.Sprintf(
			"t Yog ...and %d more tells, too many to send at once.\n",
			n-(maxTells-1)))
		chk(t, fmt.Sprintf("capTells() of %d tells", n), want,
			capTells("Yog", tells(n)))
	}
}
