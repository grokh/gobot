package main

import (
	"io/ioutil"
	"log"
	"net/smtp"
	"strings"
)

func SendBootEmail() {
	content, err := ioutil.ReadFile("tokens.txt")
	ChkErr(err)
	lines := strings.Split(string(content), "\n")

	from := lines[0]
	pwd := lines[1]
	to := lines[2 : len(lines)-1] // vim adds a newline at end of file? careful

	sub := "Subject: TorilMUD reboot/crash:\r\n\r\n"
	body := "Katumi detected a new TorilMUD boot."
	msg := []byte(sub + body)
	server := "smtp.gmail.com"
	tls := ":587"
	log.Printf("Email: %s\n", body)
	auth := smtp.PlainAuth("", from, pwd, server)
	err = smtp.SendMail(server+tls, auth, from, to, msg)
	ChkErr(err)
}
