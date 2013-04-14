package main

import (
	"log"
	"net/smtp"
)

func Notify() {
	from := "example@gmail.com"
	to := "example@gmail.com"
	body := "This is the email body."
	srv := "smtp.gmail.com:587"
	pwd := "blahblah"
	auth := smtp.PlainAuth(
		"",
		from,
		pwd,
		srv,
	)
	err := smtp.SendMail(
		srv,
		auth,
		from,
		[]string{to},
		[]byte(body),
	)
	if err != nil {
		log.Fatal(err)
	}
}
