package main

import (
	"io/ioutil"
	"strings"
)

func Identify(filename string) {
	content, err := ioutil.ReadFile(filename)
	ChkErr(err)

	// do full text processing like moving stuff onto the same line
	// put all flags/restricts, or effects, on one line
	//items = re.sub(r'(?<=[A-Z]){2}\n(?=[A-Z]{2})',' ', stats[0])
	// put enchant info on one line
	//items = re.sub(r'\n(?=Duration)',' ', items)

	items := strings.Split(string(content), "\n\n")

	for _, item := range items {
		lines := strings.Split(item, "\n")

		for _, line := range lines {
			// use regex to capture useful info
			_ = line
		}
	}
}
