package hipchat

import (
	"time"
)

const (
	ISO8601 = "2006-01-02T15:04:05-0700"
)

type Message struct {
	// Date message was sent in ISO-8601 format in request timezone.
	ISODate string `json:"date"`

	// Name and user_id of sender. user_id will be "api" for API messages and "guest" for guest messages.
	From struct {
		Name   string
		UserId interface{} `json:"user_id"`
	}

	// Message body.
	Message string

	// Name, size, and URL of uploaded file.
	File struct {
		Name string
		Size int
		URL  string
	}
}

func (m *Message) Time() (time.Time, error) {
	return time.Parse(ISO8601, m.ISODate)
}
