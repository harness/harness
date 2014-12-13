package irc

import (
	"fmt"

	"github.com/drone/drone/shared/model"
	"github.com/thoj/go-ircevent"
)

const (
	MessageStarted = "Building: %s, commit %s, author %s"
	MessageSuccess = "Success: %s, commit %s, author %s"
	MessageFailure = "Failed: %s, commit %s, author %s"
)

type IRC struct {
	Channel string
	Nick    string
	Server  string
	Started *bool `yaml:"on_started,omitempty"`
	Success *bool `yaml:"on_success,omitempty"`
	Failure *bool `yaml:"on_failure,omitempty"`
}

func (i *IRC) Send(req *model.Request) error {
	switch {
	case req.Commit.Status == model.StatusStarted && i.Started != nil && *i.Started == true:
		return i.sendStarted(req)
	case req.Commit.Status == model.StatusSuccess && i.Success != nil && *i.Success == true:
		return i.sendSuccess(req)
	case req.Commit.Status == model.StatusFailure && i.Failure != nil && *i.Failure == true:
		return i.sendFailure(req)
	}
	return nil
}

func (i *IRC) sendStarted(req *model.Request) error {
	msg := fmt.Sprintf(MessageStarted, req.Repo.Name, req.Commit.ShaShort(), req.Commit.Author)
	return i.send(i.Channel, msg)
}

func (i *IRC) sendFailure(req *model.Request) error {
	msg := fmt.Sprintf(MessageFailure, req.Repo.Name, req.Commit.ShaShort(), req.Commit.Author)
	return i.send(i.Channel, msg)
}

func (i *IRC) sendSuccess(req *model.Request) error {
	msg := fmt.Sprintf(MessageSuccess, req.Repo.Name, req.Commit.ShaShort(), req.Commit.Author)
	return i.send(i.Channel, msg)
}

// send is a helper function that will send notice messages
// to the connected IRC client
func (i *IRC) send(channel string, message string) error {
	client := irc.IRC(i.Nick, i.Nick)

	if client == nil {
		return fmt.Errorf("Error creating IRC client")
	}

	err := client.Connect(i.Server)

	if err != nil {
		return fmt.Errorf("Error connecting to IRC server: %v", err)
	}

	client.AddCallback("001", func(_ *irc.Event) {
		client.Notice(channel, message)
		client.Quit()
	})

	go client.Loop()

	return nil
}
