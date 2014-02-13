package model

import (
	"net/url"
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

	// Bitbucket Consumer Key and secret.
	BitbucketKey    string `meddler:"bitbucket_key"`
	BitbucketSecret string `meddler:"bitbucket_secret"`

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
