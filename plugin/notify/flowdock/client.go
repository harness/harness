package flowdock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	ENDPOINT = "https://api.flowdock.com/v1/messages/team_inbox/"
)

var (
	// Required default client settings
	Token       = ""
	Source      = ""
	FromAddress = ""

	// Optional default client settings
	FromName = ""
	ReplyTo  = ""
	Project  = ""
	Link     = ""
	Tags     = []string{}
)

type Client struct {
	// Required
	Token       string
	Source      string
	FromAddress string
	Subject     string
	Content     string

	// Optional
	FromName string
	ReplyTo  string
	Project  string
	Link     string
	Tags     []string
}

func (c *Client) Inbox(subject, content string) error {
	return send(c.Token, c.Source, c.FromAddress, subject, content, c.FromName, c.ReplyTo, c.Project, c.Link, c.Tags)
}

func Inbox(subject, content string) error {
	return send(Token, Source, FromAddress, subject, content, FromName, ReplyTo, Project, Link, Tags)
}

func send(token, source, fromAddress, subject, content, fromName, replyTo, project, link string, tags []string) error {
	// Required validation
	if len(token) == 0 {
		return fmt.Errorf(`"Token" is required`)
	}
	if len(source) == 0 {
		return fmt.Errorf(`"Source" is required`)
	}
	if len(fromAddress) == 0 {
		return fmt.Errorf(`"FromAddress" is required`)
	}
	if len(subject) == 0 {
		return fmt.Errorf(`"Subject" is required`)
	}

	// Build payload
	payload := map[string]interface{}{
		"source":       source,
		"from_address": fromAddress,
		"subject":      subject,
		"content":      content,
	}
	if len(fromName) > 0 {
		payload["from_name"] = fromName
	}
	if len(replyTo) > 0 {
		payload["reply_to"] = replyTo
	}
	if len(project) > 0 {
		payload["project"] = project
	}
	if len(link) > 0 {
		payload["link"] = link
	}
	if len(tags) > 0 {
		payload["tags"] = tags
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Send to Flowdock
	resp, err := http.Post(ENDPOINT+token, "application/json", bytes.NewReader(jsonPayload))
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	} else {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected response from Flowdock: %s %s", resp.Status, string(bodyBytes))
	}
}
