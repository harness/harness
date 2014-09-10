package notify

import (
	"fmt"

	"github.com/drone/drone/shared/model"
	"github.com/thoj/go-ircevent"
)

const (
	ircStartedMessage = "Building: %s, commit %s, author %s"
	ircSuccessMessage = "Success: %s, commit %s, author %s"
	ircFailureMessage = "Failed: %s, commit %s, author %s"
)

type IRC struct {
	Channel string `yaml:"channel,omitempty"`
	Nick    string `yaml:"nick,omitempty"`
	Server  string `yaml:"server,omitempty"`
	Started *bool  `yaml:"on_started,omitempty"`
	Success *bool  `yaml:"on_success,omitempty"`
	Failure *bool  `yaml:"on_failure,omitempty"`
}

func (i *IRC) Send(req *model.Request) error {
	switch {
	case req.Commit.Status == "Started" && i.Started != nil && *i.Started == true:
		return i.sendStarted(req)
	case req.Commit.Status == "Success" && i.Success != nil && *i.Success == true:
		return i.sendSuccess(req)
	case req.Commit.Status == "Failure" && i.Failure != nil && *i.Failure == true:
		return i.sendFailure(req)
	}
	return nil
}

func (i *IRC) sendStarted(req *model.Request) error {
	msg := fmt.Sprintf(ircStartedMessage, req.Repo.Name, req.Commit.ShaShort(), req.Commit.Author)
	return i.send(i.Channel, msg)
}

func (i *IRC) sendFailure(req *model.Request) error {
	msg := fmt.Sprintf(ircFailureMessage, req.Repo.Name, req.Commit.ShaShort(), req.Commit.Author)
	return i.send(i.Channel, msg)
}

func (i *IRC) sendSuccess(req *model.Request) error {
	msg := fmt.Sprintf(ircSuccessMessage, req.Repo.Name, req.Commit.ShaShort(), req.Commit.Author)
	return i.send(i.Channel, msg)
}

// send is a helper function that will send notice messages
// to the connected IRC client
func (i *IRC) send(channel string, message string) error {
	client := irc.IRC(i.Nick, i.Nick)
	if client != nil {
		return fmt.Errorf("Error creating IRC client")
	}
	defer client.Disconnect()
	client.Connect(i.Server)
	client.Notice(channel, message)
	return nil
}
