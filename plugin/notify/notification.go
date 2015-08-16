package notify

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/drone/drone/plugin/notify/email"
	"github.com/drone/drone/plugin/notify/flowdock"
	"github.com/drone/drone/plugin/notify/github"
	"github.com/drone/drone/plugin/notify/irc"
	"github.com/drone/drone/plugin/notify/katoim"
	"github.com/drone/drone/plugin/notify/webhook"
	"github.com/drone/drone/shared/model"
)

type Sender interface {
	Send(context *model.Request) error
}

// Notification stores the configuration details
// for notifying a user, or group of users,
// when their Build has completed.
type Notification struct {
	Email    *email.Email       `yaml:"email,omitempty"`
	Webhook  *webhook.Webhook   `yaml:"webhook,omitempty"`
	Hipchat  *Hipchat           `yaml:"hipchat,omitempty"`
	Irc      *irc.IRC           `yaml:"irc,omitempty"`
	Slack    *Slack             `yaml:"slack,omitempty"`
	Gitter   *Gitter            `yaml:"gitter,omitempty"`
	Flowdock *flowdock.Flowdock `yaml:"flowdock,omitempty"`
	KatoIM   *katoim.KatoIM     `yaml:"katoim,omitempty"`
	Gitlab   *Gitlab            `yaml:"gitlab,omitempty"`

	GitHub github.GitHub `yaml:"--"`
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

	// send gitter notifications
	if n.Gitter != nil {
		err := n.Gitter.Send(context)
		if err != nil {
			log.Println(err)
		}
	}

	// send gitter notifications
	if n.Flowdock != nil {
		err := n.Flowdock.Send(context)
		if err != nil {
			log.Println(err)
		}
	}

	// send kato-im notifications
	if n.KatoIM != nil {
		err := n.KatoIM.Send(context)
		if err != nil {
			log.Println(err)
		}
	}

	// send Gitlab notifications
	if n.Gitlab != nil {
		err := n.Gitlab.Send(context)
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

func getBuildUrl(context *model.Request) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s", context.Host, context.Repo.Host, context.Repo.Owner, context.Repo.Name, context.Commit.Branch, context.Commit.Sha)
}

// helper fuction to sent HTTP Post requests
// with JSON data as the payload.
func sendJson(url string, payload []byte, headers map[string]string) error {
	client := &http.Client{}
	buf := bytes.NewBuffer(payload)

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if headers != nil {
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
