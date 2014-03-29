package model

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrInvalidGitHubTrailingSlash = errors.New("GitHub URL should not have a trailing slash")
	ErrInvalidSmtpAddress         = errors.New("SMTP From Address must be provided")
	ErrInvalidSmtpPort            = errors.New("SMTP Port must be provided")
)

type Settings struct {
	ID int64 `meddler:"id,pk"`

	// SMTP settings.
	SmtpServer   string `meddler:"smtp_server"`
	SmtpPort     string `meddler:"smtp_port"`
	SmtpAddress  string `meddler:"smtp_address"`
	SmtpUsername string `meddler:"smtp_username"`
	SmtpPassword string `meddler:"smtp_password"`

	// GitHub Consumer key and secret.
	GitHubKey    string `meddler:"github_key"`
	GitHubSecret string `meddler:"github_secret"`
	GitHubDomain string `meddler:"github_domain"`
	GitHubApiUrl string `meddler:"github_apiurl"`

	// Bitbucket Consumer Key and secret.
	BitbucketKey    string `meddler:"bitbucket_key"`
	BitbucketSecret string `meddler:"bitbucket_secret"`

	// GitLab Domain
	GitlabDomain string `meddler:"gitlab_domain"`
	GitlabApiUrl string `meddler:"gitlab_apiurl"`

	// Domain of the server, eg drone.io
	Domain string `meddler:"hostname"`

	// Scheme of the server, eg https
	Scheme string `meddler:"scheme"`

	OpenInvitations bool `meddler:"open_invitations"`
}

func (s *Settings) URL() *url.URL {
	return &url.URL{
		Scheme: s.Scheme,
		Host:   s.Domain}
}

// Validate verifies all required fields are correctly populated.
func (s *Settings) Validate() error {
	switch {
	case strings.HasSuffix(s.GitHubApiUrl, "/"):
		return ErrInvalidGitHubTrailingSlash
	case len(s.SmtpServer) != 0 && len(s.SmtpPort) == 0:
		return ErrInvalidSmtpPort
	case len(s.SmtpServer) != 0 && len(s.SmtpAddress) == 0:
		return ErrInvalidSmtpAddress
	default:
		return nil
	}
}
