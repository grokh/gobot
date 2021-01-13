package main

import (
	"fmt"
	"strings"
)

func GlistStats(list string) []string {
	list = strings.Trim(list, "| ")
	list = strings.Replace(list, "|", "\n", -1)
	stats := ParseList(list)
	var txt []string
	for _, stat := range stats {
		txt = append(txt, fmt.Sprintf("%s\n", stat))
	}
	return txt
}
