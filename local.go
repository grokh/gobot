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
		if strings.Contains(stat, "404 item not found:") {
			txt = append(txt, fmt.Sprintf("%s\n", stat))
		} else {
			n := strings.Index(stat, " (")
			itemName := stat[0:n]
			n = strings.Index(stat, "Last ID:")
			idDate := stat[n : len(stat)]
			txt = append(txt, fmt.Sprintf("%s - %s\n", idDate, itemName))
		}
	}
	return txt
}
