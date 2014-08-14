package notify

import (
	"log"

	"github.com/drone/drone/plugin/notify/email"
	"github.com/drone/drone/plugin/notify/github"
	"github.com/drone/drone/shared/model"
)

type Sender interface {
	Send(context *model.Request) error
}

// Notification stores the configuration details
// for notifying a user, or group of users,
// when their Build has completed.
type Notification struct {
	Email   *email.Email `yaml:"email,omitempty"`
	Webhook *Webhook     `yaml:"webhook,omitempty"`
	Hipchat *Hipchat     `yaml:"hipchat,omitempty"`
	Irc     *IRC         `yaml:"irc,omitempty"`
	Slack   *Slack       `yaml:"slack,omitempty"`

	GitHub  github.GitHub `yaml:"--"`
}

func (n *Notification) Send(context *model.Request) error {
	// send email notifications
	if n.Email != nil {
		err := n.Email.Send(context)
		if err != nil {
			log.Println(err)
		}
	}

	// send webhook notifications
	if n.Webhook != nil {
		err := n.Webhook.Send(context)
		if err != nil {
			log.Println(err)
		}
	}

	// send hipchat notifications
	if n.Hipchat != nil {
		err := n.Hipchat.Send(context)
		if err != nil {
			log.Println(err)
		}
	}

	// send irc notifications
	if n.Irc != nil {
		err := n.Irc.Send(context)
		if err != nil {
			log.Println(err)
		}
	}

	// send slack notifications
	if n.Slack != nil {
		err := n.Slack.Send(context)
		if err != nil {
			log.Println(err)
		}
	}

	// send email notifications
	// TODO (bradrydzewski) need to improve this code
	githubStatus := new(github.GitHub)
	if err := githubStatus.Send(context); err != nil {
		log.Println(err)
	}

	return nil
}
