package notify

import (
	"encoding/json"
	"fmt"

	"github.com/drone/drone/shared/model"
)

const (
	slackStartedMessage         = "*Building* <%s|%s> (%s) by %s"
	slackStartedFallbackMessage = "Building %s (%s) by %s"
	slackSuccessMessage         = "*Success* <%s|%s> (%s) by %s"
	slackSuccessFallbackMessage = "Success %s (%s) by %s"
	slackFailureMessage         = "*Failed* <%s|%s> (%s) by %s"
	slackFailureFallbackMessage = "Failed %s (%s) by %s"
)

type Slack struct {
	WebhookUrl string `yaml:"webhook_url,omitempty"`
	Channel    string `yaml:"channel,omitempty"`
	Username   string `yaml:"username,omitempty"`
	Started    bool   `yaml:"on_started,omitempty"`
	Success    bool   `yaml:"on_success,omitempty"`
	Failure    bool   `yaml:"on_failure,omitempty"`
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
	// drone/drone#3333333
	linktext := context.Repo.Owner + "/" + context.Repo.Name + "#" + context.Commit.ShaShort()

	return fmt.Sprintf(message, url, linktext, context.Commit.Branch, context.Commit.Author)
}

func (s *Slack) getFallbackMessage(context *model.Request, message string) string {
	// drone/drone#3333333
	text := context.Repo.Owner + "/" + context.Repo.Name + "#" + context.Commit.ShaShort()

	return fmt.Sprintf(message, text, context.Commit.Branch, context.Commit.Author)
}

func (s *Slack) sendStarted(context *model.Request) error {
	return s.send(s.getMessage(context, slackStartedMessage)+"\n - "+context.Commit.Message,
		s.getFallbackMessage(context, slackStartedFallbackMessage), "warning")
}

func (s *Slack) sendSuccess(context *model.Request) error {
	return s.send(s.getMessage(context, slackSuccessMessage),
		s.getFallbackMessage(context, slackSuccessFallbackMessage), "good")
}

func (s *Slack) sendFailure(context *model.Request) error {
	return s.send(s.getMessage(context, slackFailureMessage),
		s.getFallbackMessage(context, slackFailureFallbackMessage), "danger")
}

// helper function to send HTTP requests
func (s *Slack) send(msg string, fallback string, color string) error {
	type Attachment struct {
		Fallback string   `json:"fallback"`
		Text     string   `json:"text"`
		Color    string   `json:"color"`
		MrkdwnIn []string `json:"mrkdwn_in"`
	}

	attachments := []Attachment{
		Attachment{
			fallback,
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

	return sendJson(s.WebhookUrl, payload, nil)
}
