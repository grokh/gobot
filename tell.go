package main

import (
	//"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	//"log"
	"strings"
	//"time"
)

func NotFound(four string, oper string) string {
	return "404 " + four + " not found: " + oper
}

func Reply(char string, msg string) {
	fmt.Printf("t %s %s\n", char, msg)
}

func FindItem(item string, length string) string {
	return item + length
}

func ReplyTo(char string, tell string) {
	info := "I am a Helper Bot (Beta). " +
		"Valid commands: ?, help <cmd>, hidden?, who <char>, char <char>, " +
		"clist <char>, find <char>, class <class>, delalt <char>, addalt <char>, " +
		"lr, lr <report>, stat <item>, astat <item>, fstat <att> <comp> <val>. " +
		"For further information, tell katumi help <cmd>"
	// By default, replies will use 'invalid syntax', requiring reassignment
	syntax := "Invalid syntax. For valid syntax: tell katumi ?, " +
		"tell katumi help <cmd>"
	txt := syntax

	oper := ""
	split := strings.SplitN(tell, " ", 2)
	cmd := strings.ToLower(split[0])
	if len(split) > 1 {
		oper = split[1]
	}

	// debugging
	fmt.Printf("Cmd : %s\n", cmd)
	fmt.Printf("Oper: %s\n", oper)

	if cmd == "?" {
		Reply(char, info)
	} else if cmd == "help" {
		if oper == "" {
			Reply(char, info)
		} else if oper == "?" {
			txt = "Syntax: tell katumi ? -- Katumi provides a full listing of valid commands."
			Reply(char, txt)
		} else {
			txt = NotFound("help file", oper)
			Reply(char, txt)
		}
	} else if cmd == "stat" {
		FindItem(oper, "short_stats")
	} else if cmd == "astat" {
		FindItem(oper, "long_stats")
	} else if cmd == "fstat" {
		opers := strings.Split(oper, ", ")
		query := "SELECT short_stats FROM items"
		var params []string
		for _, ops := range opers {
			fop := strings.Fields(ops)
			if len(fop) == 3 {
				fop[0] = strings.ToLower(fop[0])
				comp := fop[1]
				if strings.ContainsAny(comp, "=<>") {
					if !strings.Contains(query, "WHERE") {
						query += " WHERE item_id IN"
					} else {
						query += " AND item_id IN"
					}
					query += " (SELECT i.item_id FROM items i, item_attribs a "+
						"WHERE i.item_id = a.item_id "+
						"AND attrib_abbr = ? AND attrib_value "+comp+" ?)"
					params = append(params, fop[0], fop[2])
				}
			}
		}
	} else {
		Reply(char, syntax)
	}
}
