package notify

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	slackEndpoint       = "https://%s.slack.com/services/hooks/incoming-webhook?token=%s"
	slackStartedMessage = "*Building* %s <%s|%s>, by %s:\n> %s"
	slackSuccessMessage = "*Success* %s <%s|%s>, by %s:\n> %s"
	slackFailureMessage = "*Failed* %s <%s|%s>, by %s:\n> %s"
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

func getBuildUrl(context *Context) string {
	branchQuery := url.Values{}
	if context.Commit.Branch != "" {
		branchQuery.Set("branch", context.Commit.Branch)
	}

	return fmt.Sprintf("%s/%s/commit/%s?%s", context.Host, context.Repo.Slug, context.Commit.Hash, branchQuery.Encode())
}

func getMessage(context *Context, message string) string {
	url := getBuildUrl(context)
	return fmt.Sprintf(
		message,
		context.Repo.Name,
		url,
		context.Commit.HashShort(),
		context.Commit.Author,
		context.Commit.Message)
}

func (s *Slack) sendStarted(context *Context) error {
	return s.send(getMessage(context, slackStartedMessage), "warning")
}

func (s *Slack) sendSuccess(context *Context) error {
	return s.send(getMessage(context, slackSuccessMessage), "good")
}

func (s *Slack) sendFailure(context *Context) error {
	return s.send(getMessage(context, slackFailureMessage), "danger")
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
	go sendJson(url, payload)

	return nil
}
