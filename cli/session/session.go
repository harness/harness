// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

var (
	ErrTokenExpired = errors.New("token is expired, please login")
)

type Session struct {
	path        string
	URI         string `json:"uri"`
	ExpiresAt   int64  `json:"expires_at"`
	AccessToken string `json:"access_token"`
}

// LoadFromPath loads an existing session from a file.
func LoadFromPath(path string) (Session, error) {
	session := Session{
		path: path,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return session, fmt.Errorf("failed to read session from file: %w", err)
	}
	if err = json.Unmarshal(data, &session); err != nil {
		return session, fmt.Errorf("failed to deserialize session: %w", err)
	}

	if time.Now().Unix() > session.ExpiresAt {
		return session, ErrTokenExpired
	}

	return session, nil
}

// Store stores an existing session to the default file.
func (s Session) Store() error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to serialize session: %w", err)
	}

	err = os.WriteFile(s.path, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write session to file: %w", err)
	}

	return nil
}

func (s Session) SetURI(uri string) Session {
	s.URI = uri
	return s
}

func (s Session) SetExpiresAt(expiresAt int64) Session {
	s.ExpiresAt = expiresAt
	return s
}

func (s Session) SetAccessToken(token string) Session {
	s.AccessToken = token
	return s
}

func (s Session) Path() string {
	return s.path
}
