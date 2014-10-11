package notify

import (
	"encoding/json"
	"fmt"

	"github.com/drone/drone/shared/model"
)

const (
	slackEndpoint       = "https://%s.slack.com/services/hooks/incoming-webhook?token=%s"
	slackStartedMessage = "*Building* %s, commit <%s|%s>, author %s"
	slackSuccessMessage = "*Success* %s, commit <%s|%s>, author %s"
	slackFailureMessage = "*Failed* %s, commit <%s|%s>, author %s"
)

type Slack struct {
	Team     string `yaml:"team,omitempty"`
	Channel  string `yaml:"channel,omitempty"`
	Username string `yaml:"username,omitempty"`
	Token    string `yaml:"token,omitempty"`
	Started  bool   `yaml:"on_started,omitempty"`
	Success  bool   `yaml:"on_success,omitempty"`
	Failure  bool   `yaml:"on_failure,omitempty"`
}

func (s *Slack) Send(context *model.Request) error {
	switch {
	case context.Commit.Status == "Started" && s.Started:
		return s.sendStarted(context)
	case context.Commit.Status == "Success" && s.Success:
		return s.sendSuccess(context)
	case context.Commit.Status == "Failure" && s.Failure:
		return s.sendFailure(context)
	}

	return nil
}

func (s *Slack) getMessage(context *model.Request, message string) string {
	url := getBuildUrl(context)
	return fmt.Sprintf(message, context.Repo.Name, url, context.Commit.ShaShort(), context.Commit.Author)
}

func (s *Slack) sendStarted(context *model.Request) error {
	return s.send(s.getMessage(context, slackStartedMessage), "warning")
}

func (s *Slack) sendSuccess(context *model.Request) error {
	return s.send(s.getMessage(context, slackSuccessMessage), "good")
}

func (s *Slack) sendFailure(context *model.Request) error {
	return s.send(s.getMessage(context, slackFailureMessage), "danger")
}

// helper function to send HTTP requests
func (s *Slack) send(msg string, color string) error {
	type Attachment struct {
		Fallback string   `json:"fallback"`
		Text     string   `json:"text"`
		Color    string   `json:"color"`
		MrkdwnIn []string `json:"mrkdwn_in"`
	}

	attachments := []Attachment{
		Attachment{
			msg,
			msg,
			color,
			[]string{"fallback", "text"},
		},
	}
	// data will get posted in this format
	data := struct {
		Channel     string       `json:"channel"`
		Username    string       `json:"username"`
		Attachments []Attachment `json:"attachments"`
	}{s.Channel, s.Username, attachments}

	// data json encoded
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// send payload
	url := fmt.Sprintf(slackEndpoint, s.Team, s.Token)

	go sendJson(url, payload, nil)

	return nil
}
