package notify

import (
	"fmt"
	"strings"

	"github.com/drone/drone/shared/model"
	"github.com/stvp/flowdock"
)

const (
	flowdockStartedSubject = "Building %s (%s)"
	flowdockSuccessSubject = "Build: %s (%s) is SUCCESS"
	flowdockFailureSubject = "Build: %s (%s) is FAILED"
	flowdockMessage        = "<h2>%s </h2>\nBuild: %s <br/>\nResult: %s <br/>\nAuthor: %s <br/>Commit: <span class=\"commit-message\">%s</span> <br/>\nRepository Url: %s"
	flowdockBuildOkEmail   = "build+ok@flowdock.com"
	flowdockBuildFailEmail = "build+fail@flowdock.com"
)

type Flowdock struct {
	Token   string `yaml:"token,omitempty"`
	Source  string `yaml:"source,omitempty"`
	Tags    string `yaml:"tags,omitempty"`
	Started bool   `yaml:"on_started,omitempty"`
	Success bool   `yaml:"on_success,omitempty"`
	Failure bool   `yaml:"on_failure,omitempty"`
}

func (f *Flowdock) Send(context *model.Request) error {
	switch {
	case context.Commit.Status == "Started" && f.Started:
		return f.sendStarted(context)
	case context.Commit.Status == "Success" && f.Success:
		return f.sendSuccess(context)
	case context.Commit.Status == "Failure" && f.Failure:
		return f.sendFailure(context)
	}

	return nil
}

func (f *Flowdock) getBuildUrl(context *model.Request) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s", context.Host, context.Repo.Host, context.Repo.Owner, context.Repo.Name, context.Commit.Branch, context.Commit.Sha)
}

func (f *Flowdock) getRepoUrl(context *model.Request) string {
	return fmt.Sprintf("%s/%s/%s/%s", context.Host, context.Repo.Host, context.Repo.Owner, context.Repo.Name)
}

func (f *Flowdock) getMessage(context *model.Request) string {
	buildUrl := fmt.Sprintf("<a href=\"%s\"><span class=\"commit-sha\">%s</span></a>", f.getBuildUrl(context), context.Commit.ShaShort())
	return fmt.Sprintf(flowdockMessage, context.Repo.Name, buildUrl, context.Commit.Status, context.Commit.Author, context.Commit.Message, f.getRepoUrl(context))
}

func (f *Flowdock) sendStarted(context *model.Request) error {
	fromAddress := context.Commit.Author
	subject := fmt.Sprintf(flowdockStartedSubject, context.Repo.Name, context.Commit.Branch)
	msg := f.getMessage(context)
	tags := strings.Split(f.Tags, ",")
	return f.send(fromAddress, subject, msg, tags)
}

func (f *Flowdock) sendFailure(context *model.Request) error {
	fromAddress := flowdockBuildFailEmail
	tags := strings.Split(f.Tags, ",")
	subject := fmt.Sprintf(flowdockFailureSubject, context.Repo.Name, context.Commit.Branch)
	msg := f.getMessage(context)
	return f.send(fromAddress, subject, msg, tags)
}

func (f *Flowdock) sendSuccess(context *model.Request) error {
	fromAddress := flowdockBuildOkEmail
	tags := strings.Split(f.Tags, ",")
	subject := fmt.Sprintf(flowdockSuccessSubject, context.Repo.Name, context.Commit.Branch)
	msg := f.getMessage(context)
	return f.send(fromAddress, subject, msg, tags)
}

// helper function to send Flowdock requests
func (f *Flowdock) send(fromAddress, subject, message string, tags []string) error {
	c := flowdock.Client{Token: f.Token, Source: f.Source, FromName: "drone.io", FromAddress: fromAddress, Tags: tags}
	go c.Inbox(subject, message)
	return nil
}
