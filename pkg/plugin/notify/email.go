package notify

import (
	"fmt"
	"net/smtp"
)

type Email struct {
	Recipients []string `yaml:"recipients,omitempty"`
	Success    string   `yaml:"on_success"`
	Failure    string   `yaml:"on_failure"`

	host string // smtp host address
	port string // smtp host port
	user string // smtp username for authentication
	pass string // smtp password for authentication
	from string // smtp email address. send from this address
}

// SetServer is a function that will set the SMTP
// server location and credentials
func (e *Email) SetServer(host, port, user, pass, from string) {
	e.host = host
	e.port = port
	e.user = user
	e.pass = pass
	e.from = from
}

// Send will send an email, either success or failure,
// based on the Commit Status.
func (e *Email) Send(context *Context) error {
	switch {
	case context.Commit.Status == "Success" && e.Success != "never":
		return e.sendSuccess(context)
	case context.Commit.Status == "Failure" && e.Failure != "never":
		return e.sendFailure(context)
	}

	return nil
}

// sendFailure sends email notifications to the list of
// recipients indicating the build failed.
func (e *Email) sendFailure(context *Context) error {
	// loop through and email recipients
	/*for _, email := range e.Recipients {
		if err := mail.SendFailure(context.Repo.Slug, email, context); err != nil {
			return err
		}
	}*/
	return nil
}

// sendSuccess sends email notifications to the list of
// recipients indicating the build was a success.
func (e *Email) sendSuccess(context *Context) error {
	// loop through and email recipients
	/*for _, email := range e.Recipients {
		if err := mail.SendSuccess(context.Repo.Slug, email, context); err != nil {
			return err
		}
	}*/
	return nil
}

// send is a simple helper function to format and
// send an email message.
func (e *Email) send(to, subject, body string) error {
	// Format the raw email message body
	raw := fmt.Sprintf(emailTemplate, e.from, to, subject, body)
	auth := smtp.PlainAuth("", e.user, e.pass, e.host)
	addr := fmt.Sprintf("%s:%s", e.host, e.port)

	return smtp.SendMail(addr, auth, e.from, []string{to}, []byte(raw))
}

// text-template used to generate a raw Email message
var emailTemplate = `From: %s
To: %s
Subject: %s
MIME-version: 1.0
Content-Type: text/html; charset="UTF-8"

%s`
