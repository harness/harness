package notify

import (
	"testing"

	"github.com/andybons/hipchat"
	"github.com/drone/drone/shared/model"
)

type MockHipchatClient struct {
	Request hipchat.MessageRequest
}

func (c *MockHipchatClient) PostMessage(req hipchat.MessageRequest) error {
	c.Request = req
	return nil
}

var client = &MockHipchatClient{}

var subject = &Hipchat{
	Room:    "SampleRoom",
	Token:   "foo",
	Started: true,
	Success: true,
	Failure: true,
}

var request = &model.Request{
	Host: "http://examplehost.com",
	Repo: &model.Repo{
		Host:  "examplegit.com",
		Owner: "owner",
		Name:  "repo",
	},
	Commit: &model.Commit{
		Sha:     "abc",
		Branch:  "example",
		Status:  "Started",
		Message: "Test Commit",
		Author:  "Test User",
	},
	User: &model.User{
		Login: "TestUser",
	},
}

func Test_SendStarted(t *testing.T) {
	request.Commit.Status = "Started"

	subject.SendWithClient(client, request)
	expected := hipchat.MessageRequest{
		RoomId:        "SampleRoom",
		From:          "Drone",
		Message:       "Building <a href=\"http://examplehost.com/examplegit.com/owner/repo/example/abc\">owner/repo#abc</a> (example) by Test User <br> - Test Commit",
		Color:         hipchat.ColorYellow,
		MessageFormat: hipchat.FormatHTML,
		Notify:        false,
	}

	if client.Request != expected {
		t.Errorf("Invalid hipchat payload. Expected: %v, got %v", expected, client.Request)
	}
}

func Test_SendSuccess(t *testing.T) {
	request.Commit.Status = "Success"

	subject.SendWithClient(client, request)
	expected := hipchat.MessageRequest{
		RoomId:        "SampleRoom",
		From:          "Drone",
		Message:       "Success <a href=\"http://examplehost.com/examplegit.com/owner/repo/example/abc\">owner/repo#abc</a> (example) by Test User",
		Color:         hipchat.ColorGreen,
		MessageFormat: hipchat.FormatHTML,
		Notify:        false,
	}

	if client.Request != expected {
		t.Errorf("Invalid hipchat payload. Expected: %v, got %v", expected, client.Request)
	}
}

func Test_SendFailure(t *testing.T) {
	request.Commit.Status = "Failure"

	subject.SendWithClient(client, request)
	expected := hipchat.MessageRequest{
		RoomId:        "SampleRoom",
		From:          "Drone",
		Message:       "Failed <a href=\"http://examplehost.com/examplegit.com/owner/repo/example/abc\">owner/repo#abc</a> (example) by Test User",
		Color:         hipchat.ColorRed,
		MessageFormat: hipchat.FormatHTML,
		Notify:        true,
	}

	if client.Request != expected {
		t.Errorf("Invalid hipchat payload. Expected: %v, got %v", expected, client.Request)
	}
}
