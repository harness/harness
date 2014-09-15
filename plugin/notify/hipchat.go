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

func (h *Hipchat) Send(context *model.Request) error {
	switch {
	case context.Commit.Status == "Started" && h.Started:
		return h.sendStarted(context)
	case context.Commit.Status == "Success" && h.Success:
		return h.sendSuccess(context)
	case context.Commit.Status == "Failure" && h.Failure:
		return h.sendFailure(context)
	}

	return nil
}

func (h *Hipchat) buildLink(context *model.Request) string {
	repoName := context.Repo.Owner + "/" + context.Repo.Name
	url := context.Host + "/" + context.Repo.Host + "/" + repoName + "/" + context.Commit.Branch + "/" + context.Commit.Sha
	return fmt.Sprintf("<a href=\"%s\">%s#%s</a>", url, repoName, context.Commit.ShaShort())
}

func (h *Hipchat) sendStarted(context *model.Request) error {
	msg := fmt.Sprintf(startedMessage, h.buildLink(context), context.Commit.Branch, context.User.Login, context.Commit.Message)
	return h.send(hipchat.ColorYellow, hipchat.FormatHTML, msg, false)
}

func (h *Hipchat) sendFailure(context *model.Request) error {
	msg := fmt.Sprintf(failureMessage, h.buildLink(context), context.Commit.Branch, context.User.Login)
	return h.send(hipchat.ColorRed, hipchat.FormatHTML, msg, true)
}

func (h *Hipchat) sendSuccess(context *model.Request) error {
	msg := fmt.Sprintf(successMessage, h.buildLink(context), context.Commit.Branch, context.User.Login)
	return h.send(hipchat.ColorGreen, hipchat.FormatHTML, msg, false)
}

// helper function to send Hipchat requests
func (h *Hipchat) send(color, format, message string, notify bool) error {
	c := hipchat.Client{AuthToken: h.Token}
	req := hipchat.MessageRequest{
		RoomId:        h.Room,
		From:          "Drone",
		Message:       message,
		Color:         color,
		MessageFormat: format,
		Notify:        notify,
	}

	return c.PostMessage(req)
}
