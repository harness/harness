package notify

import (
	"fmt"

	"github.com/andybons/hipchat"
)

const (
	startedMessage = "Building %s, commit %s, author %s"
	successMessage = "<b>Success</b> %s, commit %s, author %s"
	failureMessage = "<b>Failed</b> %s, commit %s, author %s"
)

type Hipchat struct {
	Room    string `yaml:"room,omitempty"`
	Token   string `yaml:"token,omitempty"`
	Started bool   `yaml:"on_started,omitempty"`
	Success bool   `yaml:"on_success,omitempty"`
	Failure bool   `yaml:"on_failure,omitempty"`
}

func (h *Hipchat) Send(context *Context) error {
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

func (h *Hipchat) sendStarted(context *Context) error {
	msg := fmt.Sprintf(startedMessage, context.Repo.Name, context.Commit.HashShort(), context.Commit.Author)
	return h.send(hipchat.ColorYellow, hipchat.FormatHTML, msg)
}

func (h *Hipchat) sendFailure(context *Context) error {
	msg := fmt.Sprintf(failureMessage, context.Repo.Name, context.Commit.HashShort(), context.Commit.Author)
	return h.send(hipchat.ColorRed, hipchat.FormatHTML, msg)
}

func (h *Hipchat) sendSuccess(context *Context) error {
	msg := fmt.Sprintf(successMessage, context.Repo.Name, context.Commit.HashShort(), context.Commit.Author)
	return h.send(hipchat.ColorGreen, hipchat.FormatHTML, msg)
}

// helper function to send Hipchat requests
func (h *Hipchat) send(color, format, message string) error {
	c := hipchat.Client{AuthToken: h.Token}
	req := hipchat.MessageRequest{
		RoomId:        h.Room,
		From:          "Drone",
		Message:       message,
		Color:         color,
		MessageFormat: format,
		Notify:        true,
	}

	return c.PostMessage(req)
}
