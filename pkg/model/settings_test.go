package model

import (
	"testing"
)

// Ensure that defaults are properly set
func TestSetDefaults(t *testing.T) {
	s := Settings{
		SmtpPort: "",
	}

	s.SetDefaults()
	expected := "25"
	if s.SmtpPort != expected {
		t.Errorf("Expected default SMTP Port %s, got %s", expected, s.SmtpPort)
	}
}
