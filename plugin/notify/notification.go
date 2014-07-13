package notify

import (
	"github.com/drone/drone/shared/model"
)

type Sender interface {
	Send(context *model.Request) error
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

func (n *Notification) Send(context *model.Request) error {
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
