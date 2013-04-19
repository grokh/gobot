package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func Identify(filename string) {
	f, _ := os.Open("gobot.log")
	log.SetOutput(f)

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		_ = line
	}
}
