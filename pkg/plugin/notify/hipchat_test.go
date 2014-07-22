package notify

import (
	"bytes"
	"testing"
)

func Test_parseTemplate(t *testing.T) {
	tmpl := parseTemplate("test", "ok", "")
	var msg bytes.Buffer
	tmpl.Execute(&msg, interface{})
	expected = "ok"
	output := msg.String()

	if output != expected {
		t.Errorf("Failed to build url. Expected: %s, got %s", expected, output)
	}
}
