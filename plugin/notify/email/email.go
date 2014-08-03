package email

import (
	"bytes"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"

	"github.com/drone/drone/shared/model"
)

const (
	NotifyAlways = "always" // always send email notification
	NotifyNever  = "never"  // never send email notifications
	NotifyAuthor = "author" // only send email notifications to the author

	NotifyTrue  = "true"  // alias for NotifyTrue
	NotifyFalse = "false" // alias for NotifyFalse
	NotifyOn    = "on"    // alias for NotifyTrue
	NotifyOff   = "off"   // alias for NotifyFalse
	NotifyBlame = "blame" // alias for NotifyAuthor
)

const (
	SubjectSuccess = "[SUCCESS] %s/%s %s"
	SubjectFailure = "[FAILURE] %s/%s %s"
)

var (
	DefaultHost = os.Getenv("SMTP_HOST")
	DefaultPort = os.Getenv("SMTP_PORT")
	DefaultFrom = os.Getenv("SMTP_FROM")
	DefaultUser = os.Getenv("SMTP_USER")
	DefaultPass = os.Getenv("SMTP_PASS")
)

type Email struct {
	Recipients []string `yaml:"recipients"`
	Success    string   `yaml:"on_success"`
	Failure    string   `yaml:"on_failure"`

	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	From     string `yaml:"from"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Send will send an email, either success or failure,
// based on the Commit Status.
func (e *Email) Send(context *model.Request) error {
	var status = context.Commit.Status

	switch status {
	// no builds are triggered for pending builds
	case model.StatusEnqueue, model.StatusStarted:
		return nil
	case model.StatusSuccess:
		return e.sendSuccess(context)
	default:
		return e.sendFailure(context)
	}
}

// sendFailure sends email notifications to the list of
// recipients indicating the build failed.
func (e *Email) sendFailure(context *model.Request) error {

	switch e.Failure {
	case NotifyFalse, NotifyNever, NotifyOff:
		return nil
	// if configured to email the author, replace
	// the recipiends with the commit author email.
	case NotifyBlame, NotifyAuthor:
		e.Recipients = []string{context.Commit.Author}
	}

	// generate the email failure template
	var buf bytes.Buffer
	err := failureTemplate.ExecuteTemplate(&buf, "_", context)
	if err != nil {
		return err
	}

	// generate the email subject
	var subject = fmt.Sprintf(
		SubjectFailure,
		context.Repo.Owner,
		context.Repo.Name,
		context.Commit.ShaShort(),
	)

	return e.send(subject, buf.String(), e.Recipients)
}

// sendSuccess sends email notifications to the list of
// recipients indicating the build was a success.
func (e *Email) sendSuccess(context *model.Request) error {

	switch e.Success {
	case NotifyFalse, NotifyNever, NotifyOff:
		return nil
	// if configured to email the author, replace
	// the recipiends with the commit author email.
	case NotifyBlame, NotifyAuthor:
		e.Recipients = []string{context.Commit.Author}
	}

	// generate the email success template
	var buf bytes.Buffer
	err := successTemplate.ExecuteTemplate(&buf, "_", context)
	if err != nil {
		return err
	}

	// generate the email subject
	var subject = fmt.Sprintf(
		SubjectSuccess,
		context.Repo.Owner,
		context.Repo.Name,
		context.Commit.ShaShort(),
	)

	return e.send(subject, buf.String(), e.Recipients)
}

func (e *Email) send(subject, body string, recipients []string) error {

	if len(recipients) == 0 {
		return nil
	}

	// the user can provide their own smtp server
	// configuration. If None provided, attempt to
	// use the global configuration set in the environet
	// variables.
	if len(DefaultHost) != 0 {
		e.Host = DefaultHost
		e.Port = DefaultPort
		e.From = DefaultFrom
		e.Username = DefaultUser
		e.Password = DefaultPass
	}

	var auth smtp.Auth
	var addr = net.JoinHostPort(e.Host, e.Port)

	// setup the authentication to the smtp server
	// if the username and password are provided.
	if len(e.Username) > 0 {
		auth = smtp.PlainAuth("", e.Username, e.Password, e.Host)
	}

	// genereate the raw email message
	var to = strings.Join(e.Recipients, ",")
	var raw = fmt.Sprintf(rawMessage, e.From, to, subject, body)

	return smtp.SendMail(addr, auth, e.From, e.Recipients, []byte(raw))
}
