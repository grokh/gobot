package main

import (
	"io/ioutil"
	"regexp"
	"strconv"
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

	// initialize regex checks
	var m [][]string
	chkName, err := regexp.Compile(
		// Name 'a huge boar skull'
		`Name '([[:print:]]+)'`)
	ChkErr(err)
	chkKey, err := regexp.Compile(
		// Keyword 'skull boar', Item type: ARMOR
		`Keyword '([[:print:]]+)', Item type: ([[:word:]]+)`)
	ChkErr(err)
	chkWorn, err := regexp.Compile(
		// Item can be worn on:  HEAD 
		`Item can be worn on:  ([[:print:]]+) `)
	ChkErr(err)
	chkEff, err := regexp.Compile(
		// Item will give you following abilities:  NOBITS
		`Item will give you following abilities:  ([[:print:]]+)`)
	ChkErr(err)
	chkFlag, err := regexp.Compile(
		// Item is: NOBITSNOBITS
		`Item is: ([[:print:]]+)`)
	ChkErr(err)
	chkRest, err := regexp.Compile(
		// Item is: NOBITSNOBITS
		`[NO|ANTI]-`)
	ChkErr(err)
	chkWtval, err := regexp.Compile(
		// Weight: 2, Value: 0
		`Weight: ([[:digit:]]+), Value: ([[:digit:]]+)`)
	ChkErr(err)
	chkAC, err := regexp.Compile(
		// AC-apply is 8
		`AC-apply is ([[:digit:]]+)`)
	ChkErr(err)
	chkAttr, err := regexp.Compile(
		//     Affects : HITROLL By 2
		`    Affects : ([[:print:]]+) [B|b]y ([[:digit:]-]+)`)
	ChkErr(err)

	for _, item := range items {
		// initialize item variables
		full_stats, item_name, keywords, item_type := "", "", "", ""
		weight, c_value := -1, -1
		var item_slots, item_effects, flags, item_flags, item_restricts []string
		var item_attribs, item_specials [][]string

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
				item_slots = strings.Fields(m[0][1])
			case chkEff.MatchString(line):
				m = chkEff.FindAllStringSubmatch(line, -1)
				item_effects = strings.Fields(m[0][1])
			case chkFlag.MatchString(line):
				m = chkFlag.FindAllStringSubmatch(line, -1)
				flags = strings.Fields(m[0][1])
				for _, flag := range flags {
					if chkRest.MatchString(flag) {
						item_restricts = append(item_restricts, flag)
					} else {
						item_flags = append(item_flags, flag)
					}
				}
			case chkWtval.MatchString(line):
				m = chkWtval.FindAllStringSubmatch(line, -1)
				weight, err = strconv.Atoi(m[0][1])
				ChkErr(err)
				c_value, err = strconv.Atoi(m[0][2])
				ChkErr(err)
			case chkAC.MatchString(line):
				m = chkAC.FindAllStringSubmatch(line, -1)
				item_specials = append(item_specials, []string{item_type, "ac", m[0][1]})
			case chkAttr.MatchString(line):
				m = chkAttr.FindAllStringSubmatch(line, -1)
				item_attribs = append(item_attribs, []string{m[0][1], m[0][2]})
			}
		}
		// back to full item
		fmt.Printf("Name: %s, Keywords: %s, Type: %s\n", 
			item_name, keywords, item_type)
		fmt.Printf("Weight: %d, Value: %d\n", weight, c_value)
		for _, slot := range item_slots {
			fmt.Printf("Slot: %s\n", slot)
		}
		for _, eff := range item_effects {
			if eff != "NOBITS" && eff != "GROUP_CACHED" {
				fmt.Printf("Effect: %s\n", eff)
			}
		}
		for _, flag := range item_flags {
			if flag != "NOBITS" && flag != "NOBITSNOBITS" {
				fmt.Printf("Flag: %s\n", flag)
			}
		}
		for _, rest := range item_restricts {
			fmt.Printf("Restrict: %s\n", rest)
		}
		_ = full_stats
	}
}
