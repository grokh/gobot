package main

import (
	"io/ioutil"
	"strings"
)

func Identify(filename string) {
	content, err := ioutil.ReadFile(filename)
	ChkErr(err)

	// do full text processing like moving stuff onto the same line
	// split into items on double newline
	items := strings.Split(string(content), "\n\n")

	for _, item := range items {
		// split onto separate lines for regex checking
		lines := strings.Split(item, "\n")

		for _, line := range lines {
			_ = line
		}
	}
}
