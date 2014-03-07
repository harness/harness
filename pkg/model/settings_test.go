package model

import (
	"testing"
)

func Test_SettingsValidate(t *testing.T) {
	settings := Settings{}
	settings.GitHubApiUrl = "https://github.com/url/with/slash/"
	if err := settings.Validate(); err != ErrInvalidGitHubTrailingSlash {
		t.Errorf("Expecting ErrInvalidGitHubTrailingSlash")
	}

	settings = Settings{}
	settings.SmtpServer = "127.1.1.1"
	if err := settings.Validate(); err != ErrInvalidSmtpPort {
		t.Errorf("Expecting ErrInvalidSmtpPort")
	}

	settings = Settings{}
	settings.SmtpServer = "127.1.1.1"
	settings.SmtpPort = "553"
	if err := settings.Validate(); err != ErrInvalidSmtpAddress {
		t.Errorf("Expecting ErrInvalidSmtpAddress")
	}

	settings = Settings{}
	settings.SmtpServer = "127.1.1.1"
	settings.SmtpPort = "553"
	settings.SmtpAddress = "test@localhost"
	settings.GitHubApiUrl = "https://api.github.com"
	if err := settings.Validate(); err != nil {
		t.Errorf("Expecting successful Settings validation, got %s", err)
	}
}
