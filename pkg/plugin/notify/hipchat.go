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
		CommitAuthor: context.Commit.Author,
		RepoName:     context.Repo.Name,
	}
	var message string

	switch {
	case context.Commit.Status == "Started" && h.Started:
		message = renderMessage(hipchatContext, h.StartedMessage, startedMessage)
		return h.send(hipchat.ColorYellow, message)
	case context.Commit.Status == "Success" && h.Success:
		message = renderMessage(hipchatContext, h.SuccessMessage, successMessage)
		return h.send(hipchat.ColorGreen, message)
	case context.Commit.Status == "Failure" && h.Failure:
		message = renderMessage(hipchatContext, h.FailureMessage, failureMessage)
		return h.send(hipchat.ColorRed, message)
	}

	return nil
}

// helper function to send Hipchat requests
func (h *Hipchat) send(color, message string) error {
	c := hipchat.Client{AuthToken: h.Token}
	req := hipchat.MessageRequest{
		RoomId:        h.Room,
		From:          "Drone",
		Message:       message,
		Color:         color,
		MessageFormat: hipchat.FormatHTML,
		Notify:        true,
	}

	return c.PostMessage(req)
}

func renderMessage(context *HipchatContext, msgTmpl, defaultTmpl string) string {
	var msg bytes.Buffer
	tmpl := parseTemplate("started", msgTmpl, defaultTmpl)
	tmpl.Execute(&msg, context)
	return msg.String()
}

func parseTemplate(name, templ, def string) *template.Template {
	if templ != "" {
		tmpl, err := template.New("failure").Parse(templ)
		if err != nil {
			return parseTemplate(name, "Error:"+err.Error(), "")
		}
		return tmpl
	} else {
		return parseTemplate(name, def, "")
	}
}
