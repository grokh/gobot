package main

import (
	"io/ioutil"
	"log"
	"strings"
)

func Identify(filename string) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		_ = line
	}
}
