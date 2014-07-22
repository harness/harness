package notify

import (
	"bytes"
	"text/template"

	"github.com/andybons/hipchat"
)

const (
	startedMessage = "Building {{.RepoName}}, commit {{.CommitHash}}, author {{.CommitAuthor}}"
	successMessage = "<b>Success</b> {{.RepoName}}, commit {{.CommitHash}}, author {{.CommitAuthor}}"
	failureMessage = "<b>Failed</b> {{.RepoName}}, commit {{.CommitHash}}, author {{.CommitAuthor}}"
)

type Hipchat struct {
	Room           string `yaml:"room,omitempty"`
	Token          string `yaml:"token,omitempty"`
	Started        bool   `yaml:"on_started,omitempty"`
	Success        bool   `yaml:"on_success,omitempty"`
	Failure        bool   `yaml:"on_failure,omitempty"`
	StartedMessage string `yaml:"started_message,omitempty"`
	SuccessMessage string `yaml:"success_message,omitempty"`
	FailureMessage string `yaml:"failure_message,omitempty"`
}

type HipchatContext struct {
	CommitHash   string
	CommitAuthor string
	RepoName     string
	Host         string
}

func (h *Hipchat) Send(context *Context) error {
	hipchatContext := &HipchatContext{
		CommitHash:   context.Commit.HashShort(),
		CommitAuthor: context.CommitAuthor,
		RepoName:     context.Repo.Name,
	}
	switch {
	case context.Commit.Status == "Started" && h.Started:
		return h.sendStarted(hipchatContext)
	case context.Commit.Status == "Success" && h.Success:
		return h.sendSuccess(hipchatContext)
	case context.Commit.Status == "Failure" && h.Failure:
		return h.sendFailure(hipchatContext)
	}

	return nil
}

func (h *Hipchat) sendStarted(context *HipchatContext) error {
	var msg bytes.Buffer
	tmpl = parseTemplate("started", h.StartedMessage, startedMessage)
	tmpl.Execute(&msg, context)
	return h.send(hipchat.ColorYellow, hipchat.FormatHTML, msg.String())
}

func (h *Hipchat) sendFailure(context *HipchatContext) error {
	var msg bytes.Buffer
	tmpl = parseTemplate("failure", h.FailureMessage, failureMessage)
	tmpl.Execute(&msg, context)
	return h.send(hipchat.ColorRed, hipchat.FormatHTML, msg.String())
}

func (h *Hipchat) sendSuccess(context *HipchatContext) error {
	var msg bytes.Buffer
	tmpl = parseTemplate("success", h.SuccessMessage, successMessage)
	tmpl.Execute(&msg, context)
	return h.send(hipchat.ColorGreen, hipchat.FormatHTML, msg.String())
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

func parseTemplate(name, templ, def string) *Template {
	if templ != nil {
		tmpl, err := template.New("failure").Parse(templ)
		if err != nil {
			return parseTemplate(name, "Error:"+err.Error(), "")
		}
		return tmpl
	} else {
		return parseTemplate(name, def, "")
	}
}
