package gomail

import (
	"selarashomeid/internal/config"
	"strconv"

	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
)

func SendMail(recipient string) error {
	mailer := gomail.NewMessage()
	mailer.SetHeader("From", config.Get().Gomail.SenderName)
	mailer.SetHeader("To", recipient)
	mailer.SetAddressHeader("Cc", "tralalala@gmail.com", "Tra Lala La")
	mailer.SetHeader("Subject", "Test mail")
	mailer.SetBody("text/html", "Hello, <b>have a nice day</b>")

	portMail, _ := strconv.Atoi(config.Get().Gomail.SmtpPort)
	dialer := gomail.NewDialer(
		config.Get().Gomail.SmtpHost,
		portMail,
		config.Get().Gomail.AuthEmail,
		config.Get().Gomail.AuthPassword,
	)

	err := dialer.DialAndSend(mailer)
	if err != nil {
		return err
	}

	logrus.Info("Mail sent!")
	return nil
}
