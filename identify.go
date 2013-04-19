package main

import (
	"io/ioutil"
	"strings"
)

func Identify(filename string) {
	content, err := ioutil.ReadFile(filename)
	ChkErr(err)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		_ = line
	}
}
