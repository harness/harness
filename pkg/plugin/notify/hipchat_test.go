package notify

import (
	"bytes"
	"testing"
)

func Test_renderMessage(t *testing.T) {
	context := &HipchatContext{
		CommitHash:   "HASH",
		CommitAuthor: "drone",
		RepoName:     "drone",
		Host:         "host",
	}

	expected = "HASH drone/drone host"
	template = "{{.CommitHash}} {{.CommitAuthor}}/{{.RepoName}} {{.Host}}"
	output := renderMessage(context, template, "")

	if output != expected {
		t.Errorf("Failed to render message. Expected: %s, got %s", expected, output)
	}
}

func Test_parseTemplate(t *testing.T) {
	tmpl := parseTemplate("test", "ok", "")
	var msg bytes.Buffer
	tmpl.Execute(&msg, interface{})
	expected = "ok"
	output := msg.String()

	if output != expected {
		t.Errorf("Failed to parse template. Expected: %s, got %s", expected, output)
	}
}
