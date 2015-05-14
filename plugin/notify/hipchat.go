package notify

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/drone/drone/shared/model"
)

const (
	startedMessage = "Building %s (%s) by %s <br> - %s"
	successMessage = "Success %s (%s) by %s"
	failureMessage = "Failed %s (%s) by %s"
	defaultServer  = "api.hipchat.com"
)

type Hipchat struct {
	Server  string `yaml:"server,omitempty"`
	Room    string `yaml:"room,omitempty"`
	Token   string `yaml:"token,omitempty"`
	Started bool   `yaml:"on_started,omitempty"`
	Success bool   `yaml:"on_success,omitempty"`
	Failure bool   `yaml:"on_failure,omitempty"`
}

func (h *Hipchat) Send(context *model.Request) error {
	client := new(HipchatSimpleHTTPClient)
	return h.SendWithClient(client, context)
}

func (h *Hipchat) SendWithClient(client HipchatClient, context *model.Request) error {
	switch {
	case context.Commit.Status == "Started" && h.Started:
		return h.sendStarted(client, context)
	case context.Commit.Status == "Success" && h.Success:
		return h.sendSuccess(client, context)
	case context.Commit.Status == "Failure" && h.Failure:
		return h.sendFailure(client, context)
	}
	return nil
}

func (h *Hipchat) buildLink(context *model.Request) string {
	repoName := context.Repo.Owner + "/" + context.Repo.Name
	url := context.Host + "/" + context.Repo.Host + "/" + repoName + "/" + context.Commit.Branch + "/" + context.Commit.Sha
	return fmt.Sprintf("<a href=\"%s\">%s#%s</a>", url, repoName, context.Commit.ShaShort())
}

func (h *Hipchat) sendStarted(client HipchatClient, context *model.Request) error {
	msg := fmt.Sprintf(startedMessage, h.buildLink(context), context.Commit.Branch, context.Commit.Author, context.Commit.Message)
	req := HipchatMessageRequest{
		Server:    h.Server,
		RoomId:    h.Room,
		AuthToken: h.Token,
		Color:     "yellow",
		Message:   msg,
		Notify:    false,
	}
	return client.PostMessage(req)
}

func (h *Hipchat) sendFailure(client HipchatClient, context *model.Request) error {
	msg := fmt.Sprintf(failureMessage, h.buildLink(context), context.Commit.Branch, context.Commit.Author)
	req := HipchatMessageRequest{
		Server:    h.Server,
		RoomId:    h.Room,
		AuthToken: h.Token,
		Color:     "red",
		Message:   msg,
		Notify:    true,
	}
	return client.PostMessage(req)
}

func (h *Hipchat) sendSuccess(client HipchatClient, context *model.Request) error {
	msg := fmt.Sprintf(successMessage, h.buildLink(context), context.Commit.Branch, context.Commit.Author)
	req := HipchatMessageRequest{
		Server:    h.Server,
		RoomId:    h.Room,
		AuthToken: h.Token,
		Color:     "green",
		Message:   msg,
		Notify:    false,
	}
	return client.PostMessage(req)
}

// HipChat client

type HipchatClient interface {
	PostMessage(req HipchatMessageRequest) error
}

type HipchatMessageRequest struct {
	Server    string
	RoomId    string
	Color     string
	Message   string
	Notify    bool
	AuthToken string
}

type HipchatSimpleHTTPClient struct{}

func (*HipchatSimpleHTTPClient) PostMessage(req HipchatMessageRequest) error {
	var server string
	if len(req.Server) > 0 {
		server = req.Server
	} else {
		server = defaultServer
	}
	hipchat_uri := fmt.Sprintf("https://%s/v2/room/%s/notification", server, req.RoomId)
	_, err := http.PostForm(hipchat_uri,
		url.Values{
			"color":          {req.Color},
			"message":        {req.Message},
			"notify":         {strconv.FormatBool(req.Notify)},
			"message_format": {"html"},
			"auth_token":     {req.AuthToken},
		})
	return err
}
