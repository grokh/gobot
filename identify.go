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
	
	// put all flags/restricts, or effects, on one line
	re, err := regexp.Compile(`([[:upper:]]{2})\n([[:upper:]]{2})`)
	ChkErr(err)
	text = re.ReplaceAllString(text, "$1 $2")

	// put enchant info on one line
	re, err = regexp.Compile(`\n(Duration)`)
	ChkErr(err)
	text = re.ReplaceAllString(text, " $1")

	// put all resists on same line:
	re, err = regexp.Compile(
		`\n(    [[:alpha:] ]{6}:[[:blank:]]{3,4}[[:digit:]]{1,2}% )`)
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

	items := strings.Split(text, "\n\n")

	// initialize item variables
	full_stats, item_name, keywords, item_type, slot := "", "", "", "", ""

	// initialize regex checks
	var m [][]string
	chkName, err := regexp.Compile(
		`Name '([[:print:]]+)'`)
	ChkErr(err)
	chkKey, err := regexp.Compile(
		`Keyword '([[:print:]]+)', Item type: ([[:word:]]+)`)
	ChkErr(err)
	chkWorn, err := regexp.Compile(
		`Item can be worn on:  ([[:print:]]+) `)
	ChkErr(err)

	for _, item := range items {
		full_stats = item
		lines := strings.Split(item, "\n")

		for _, line := range lines {
			switch {
			case chkName.MatchString(line):
				m = chkName.FindAllStringSubmatch(line, -1)
				item_name = m[0][1]
			case chkKey.MatchString(line):
				m = chkKey.FindAllStringSubmatch(line, -1)
				keywords = m[0][1]
				item_type = m[0][2]
			case chkWorn.MatchString(line):
				m = chkWorn.FindAllStringSubmatch(line, -1)
				slot = m[0][1]
			}
		}
		// back to item
		fmt.Printf("Name: %s, Keywords: %s, Type: %s, Slot: %s\n", 
			item_name, keywords, item_type, slot)
		_ = full_stats
	}
}
