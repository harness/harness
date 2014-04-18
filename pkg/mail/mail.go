package mail

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"

	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/template"
)

// A Message represents an email message. Addresses may be of any
// form permitted by RFC 822.
type Message struct {
	Sender  string
	ReplyTo string // may be empty

	To      string
	Subject string
	Body    string
}

// Sends a activation email to the User.
func SendActivation(to string, data interface{}) error {
	msg := Message{}
	msg.Subject = "[drone.io] Account Activation"
	msg.To = to

	var buf bytes.Buffer
	err := template.ExecuteTemplate(&buf, "activation.html", &data)
	if err != nil {
		log.Print(err)
		return err
	}
	msg.Body = buf.String()

	return Send(&msg)
}

// Sends an invitation to join a Team
func SendInvitation(team, to string, data interface{}) error {
	msg := Message{}
	msg.Subject = "Invited to join " + team
	msg.To = to

	var buf bytes.Buffer
	err := template.ExecuteTemplate(&buf, "invitation.html", &data)
	if err != nil {
		log.Print(err)
		return err
	}
	msg.Body = buf.String()

	return Send(&msg)
}

// Sends an email to the User's email address
// with Password reset information.
func SendPassword(to string, data interface{}) error {
	msg := Message{}
	msg.Subject = "[drone.io] Password Reset"
	msg.To = to

	var buf bytes.Buffer
	err := template.ExecuteTemplate(&buf, "reset_password.html", &data)
	if err != nil {
		log.Print(err)
		return err
	}
	msg.Body = buf.String()

	return Send(&msg)
}

// Sends a build success email to the user.
func SendSuccess(repo, sha, to string, data interface{}) error {
	msg := Message{}
	msg.Subject = fmt.Sprintf("[%s] SUCCESS building %s", repo, sha)
	msg.To = to

	var buf bytes.Buffer
	err := template.ExecuteTemplate(&buf, "success.html", &data)
	if err != nil {
		log.Print(err)
		return err
	}
	msg.Body = buf.String()

	return Send(&msg)
}

// Sends a build failure email to the user.
func SendFailure(repo, sha, to string, data interface{}) error {
	msg := Message{}
	msg.Subject = fmt.Sprintf("[%s] FAILURE building %s", repo, sha)
	msg.To = to

	var buf bytes.Buffer
	err := template.ExecuteTemplate(&buf, "failure.html", &data)
	if err != nil {
		log.Print(err)
		return err
	}
	msg.Body = buf.String()

	return Send(&msg)
}

// Send sends an email message.
func Send(msg *Message) error {
	// retieve the system settings from the database
	// so that we can get the SMTP details.
	s, err := database.GetSettings()
	if err != nil {
		log.Print(err)
		return err
	}

	// set the FROM address
	msg.Sender = s.SmtpAddress

	// format the raw email message body
	body := fmt.Sprintf(emailTemplate, msg.Sender, msg.To, msg.Subject, msg.Body)

	var auth smtp.Auth
	if len(s.SmtpUsername) > 0 {
		auth = smtp.PlainAuth("", s.SmtpUsername, s.SmtpPassword, s.SmtpServer)
	}
	addr := fmt.Sprintf("%s:%s", s.SmtpServer, s.SmtpPort)

	err = smtp.SendMail(addr, auth, msg.Sender, []string{msg.To}, []byte(body))
	if err != nil {
		log.Print(err)
		return err
	}

	return nil
}

// Text-template used to generate a raw Email message
var emailTemplate = `From: %s
To: %s
Subject: %s
MIME-version: 1.0
Content-Type: text/html; charset="UTF-8"

%s`
