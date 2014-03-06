package notify

import (
	"github.com/drone/drone/pkg/model"
)

// Context represents the context of an
// in-progress build request.
type Context struct {
	// Global settings
	Host string

	// User that owns the repository
	User *model.User

	// Repository being built.
	Repo *model.Repo

	// Commit being built
	Commit *model.Commit
}

type Sender interface {
	Send(context *Context) error
}

// Notification stores the configuration details
// for notifying a user, or group of users,
// when their Build has completed.
type Notification struct {
	Email   *Email   `yaml:"email,omitempty"`
	Webhook *Webhook `yaml:"webhook,omitempty"`
	Hipchat *Hipchat `yaml:"hipchat,omitempty"`
	Irc     *IRC     `yaml:"irc,omitempty"`
}

func (n *Notification) Send(context *Context) error {
	// send email notifications
	if n.Email != nil {
		n.Email.Send(context)
	}

	// send email notifications
	if n.Webhook != nil {
		n.Webhook.Send(context)
	}

	// send email notifications
	if n.Hipchat != nil {
		n.Hipchat.Send(context)
	}

	// send irc notifications
	if n.Irc != nil {
		n.Irc.Send(context)
	}

	return nil
}
