package main

import (
	"io/ioutil"
	"regexp"
	"strings"
	"fmt"
)

func Identify(filename string) {
	content, err := ioutil.ReadFile(filename)
	ChkErr(err)
	text := string(content)

	// example: re, err := regexp.Compile(`^\[[ ]?(\d{1,2}) ([[:alpha:]-]{3})\] ([[:alpha:]]+) .*\((.*)\)`)
	
	// put all flags/restricts, or effects, on one line
	re, err := regexp.Compile(`([[:upper:]]{2})\n([[:upper:]]{2})`)
	ChkErr(err)
	text = re.ReplaceAllString(text, "$1 $2")

	// put enchant info on one line
	re, err = regexp.Compile(`\n(Duration)`)
	ChkErr(err)
	text = re.ReplaceAllString(text, " $1")

	// put all resists on same line:
	re, err = regexp.Compile(`\n(    [[:alpha:] ]{6}:[[:blank:]]{3,4}[[:digit:]]{1,2}%)`)
	ChkErr(err)
	text = re.ReplaceAllString(text, "$1")

	// put first wand/staff/scroll/potion spell on one line:
	re, err = regexp.Compile(`(spell[s]? of:)\n`)
	ChkErr(err)
	text = re.ReplaceAllString(text, "$1 ")

	// put remaining scroll/potion spells on same line:
	re, err = regexp.Compile(`\n([[:lower:]])`)
	ChkErr(err)
	text = re.ReplaceAllString(text, ", $1")

	fmt.Println(text) // debugging

	items := strings.Split(text, "\n\n")

	for _, item := range items {
		lines := strings.Split(item, "\n")

		for _, line := range lines {
			// use regex to capture useful info
			_ = line
		}
	}
}
