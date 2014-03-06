package notify

import (
	"fmt"

	irc "github.com/fluffle/goirc/client"
)

const (
	ircStartedMessage = "Building: %s, commit %s, author %s"
	ircSuccessMessage = "Success: %s, commit %s, author %s"
	ircFailureMessage = "Failed: %s, commit %s, author %s"
)

type IRC struct {
	Channel       string `yaml:"channel,omitempty"`
	Nick          string `yaml:"nick,omitempty"`
	Server        string `yaml:"server,omitempty"`
	Started       bool   `yaml:"on_started,omitempty"`
	Success       bool   `yaml:"on_success,omitempty"`
	Failure       bool   `yaml:"on_failure,omitempty"`
	SSL           bool   `yaml:"ssl,omitempty"`
	ClientStarted bool
	Client        *irc.Conn
}

func (i *IRC) Connect() {
	c := irc.SimpleClient(i.Nick)
	c.SSL = i.SSL
	connected := make(chan bool)
	c.AddHandler(irc.CONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			conn.Join(i.Channel)
			connected <- true
		})
	c.Connect(i.Server)
	<-connected
	i.ClientStarted = true
	i.Client = c
}

func (i *IRC) Send(context *Context) error {
	switch {
	case context.Commit.Status == "Started" && i.Started:
		return i.sendStarted(context)
	case context.Commit.Status == "Success" && i.Success:
		return i.sendSuccess(context)
	case context.Commit.Status == "Failure" && i.Failure:
		return i.sendFailure(context)
	}
	return nil
}

func (i *IRC) sendStarted(context *Context) error {
	msg := fmt.Sprintf(ircStartedMessage, context.Repo.Name, context.Commit.HashShort(), context.Commit.Author)
	i.send(i.Channel, msg)
	return nil
}

func (i *IRC) sendFailure(context *Context) error {
	msg := fmt.Sprintf(ircFailureMessage, context.Repo.Name, context.Commit.HashShort(), context.Commit.Author)
	i.send(i.Channel, msg)
	if i.ClientStarted {
		i.Client.Quit()
	}
	return nil
}

func (i *IRC) sendSuccess(context *Context) error {
	msg := fmt.Sprintf(ircSuccessMessage, context.Repo.Name, context.Commit.HashShort(), context.Commit.Author)
	i.send(i.Channel, msg)
	if i.ClientStarted {
		i.Client.Quit()
	}
	return nil
}

func (i *IRC) send(channel string, message string) error {
	if !i.ClientStarted {
		i.Connect()
	}
	i.Client.Notice(channel, message)
	return nil
}
