package notify

import (
	"fmt"

	"github.com/andybons/hipchat"
	"github.com/drone/drone/shared/model"
)

const (
	startedMessage = "Building %s (%s) by %s <br> - %s"
	successMessage = "Success %s (%s) by %s"
	failureMessage = "Failed %s (%s) by %s"
)

type Hipchat struct {
	Room    string `yaml:"room,omitempty"`
	Token   string `yaml:"token,omitempty"`
	Started bool   `yaml:"on_started,omitempty"`
	Success bool   `yaml:"on_success,omitempty"`
	Failure bool   `yaml:"on_failure,omitempty"`
}

type HipchatClient interface {
	PostMessage(req hipchat.MessageRequest) error
}

func (h *Hipchat) Send(context *model.Request) error {
	client := &hipchat.Client{AuthToken: h.Token}
	return h.SendWithClient(client, context)
}

func (h *Hipchat) SendWithClient(client HipchatClient, context *model.Request) error {
	switch {
	case context.Commit.Status == "Started" && h.Started:
		return h.sendStarted(client, context)
	case context.Commit.Status == "Success" && h.Success:
		return h.sendSuccess(client, context)
	case context.Commit.Status == "Failure" && h.Failure:
		return h.sendFailure(client, context)
	}

	return nil
}

func (h *Hipchat) buildLink(context *model.Request) string {
	repoName := context.Repo.Owner + "/" + context.Repo.Name
	url := context.Host + "/" + context.Repo.Host + "/" + repoName + "/" + context.Commit.Branch + "/" + context.Commit.Sha
	return fmt.Sprintf("<a href=\"%s\">%s#%s</a>", url, repoName, context.Commit.ShaShort())
}

func (h *Hipchat) sendStarted(client HipchatClient, context *model.Request) error {
	msg := fmt.Sprintf(startedMessage, h.buildLink(context), context.Commit.Branch, context.Commit.Author, context.Commit.Message)
	return h.send(client, hipchat.ColorYellow, hipchat.FormatHTML, msg, false)
}

func (h *Hipchat) sendFailure(client HipchatClient, context *model.Request) error {
	msg := fmt.Sprintf(failureMessage, h.buildLink(context), context.Commit.Branch, context.Commit.Author)
	return h.send(client, hipchat.ColorRed, hipchat.FormatHTML, msg, true)
}

func (h *Hipchat) sendSuccess(client HipchatClient, context *model.Request) error {
	msg := fmt.Sprintf(successMessage, h.buildLink(context), context.Commit.Branch, context.Commit.Author)
	return h.send(client, hipchat.ColorGreen, hipchat.FormatHTML, msg, false)
}

// helper function to send Hipchat requests
func (h *Hipchat) send(client HipchatClient, color, format, message string, notify bool) error {
	req := hipchat.MessageRequest{
		RoomId:        h.Room,
		From:          "Drone",
		Message:       message,
		Color:         color,
		MessageFormat: format,
		Notify:        notify,
	}

	return client.PostMessage(req)
}
