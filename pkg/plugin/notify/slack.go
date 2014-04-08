package notify

import (
	"encoding/json"
	"fmt"

	//"github.com/drone/drone/pkg/model"
)

const (
	slackEndpoint       = "https://%s.slack.com/services/hooks/incoming-webhook?token=%s"
	slackStartedMessage = "*Building* %s, commit %s, author %s"
	slackSuccessMessage = "*Success* %s, commit %s, author %s"
	slackFailureMessage = "*Failed* %s, commit %s, author %s"
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

func (s *Slack) Send(context *Context) error {
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

func (s *Slack) sendStarted(context *Context) error {
	msg := fmt.Sprintf(slackStartedMessage, context.Repo.Name, context.Commit.HashShort(), context.Commit.Author)
	return s.send(msg)
}

func (s *Slack) sendSuccess(context *Context) error {
	msg := fmt.Sprintf(slackSuccessMessage, context.Repo.Name, context.Commit.HashShort(), context.Commit.Author)
	return s.send(msg)
}

func (s *Slack) sendFailure(context *Context) error {
	msg := fmt.Sprintf(slackFailureMessage, context.Repo.Name, context.Commit.HashShort(), context.Commit.Author)
	return s.send(msg)
}

// helper function to send HTTP requests
func (s *Slack) send(msg string) error {
	// data will get posted in this format
	data := struct {
		Channel  string `json:"channel"`
		Username string `json:"username"`
		Text     string `json:"text"`
	}{s.Channel, s.Username, msg}

	// data json encoded
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// send payload
	url := fmt.Sprintf(slackEndpoint, s.Team, s.Token)
	go sendJson(url, payload)

	return nil
}
