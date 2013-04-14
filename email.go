package main

import (
	"log"
	"net/smtp"
)

func Notify(from string, to []string, pwd string) {
	sub := "Subject: TorilMUD reboot/crash:\r\n\r\n"
	body := "Katumi detected a new TorilMUD boot."
	msg := []byte(sub + body)
	server := "smtp.gmail.com"
	tls := ":587"
	auth := smtp.PlainAuth("", from, pwd, server)
	err := smtp.SendMail(server+tls, auth, from, to, msg)
	if err != nil {
		log.Fatal(err)
	}
}
