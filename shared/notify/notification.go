package notify

import (
	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/resource/user"
)

// Context represents the context of an
// in-progress build request.
type Context struct {
	// Global settings
	Host string

	// User that owns the repository
	User *user.User

	// Repository being built.
	Repo *repo.Repo

	// Commit being built
	Commit *commit.Commit
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
	Slack   *Slack   `yaml:"slack,omitempty"`
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

	// send slack notifications
	if n.Slack != nil {
		n.Slack.Send(context)
	}

	return nil
}
