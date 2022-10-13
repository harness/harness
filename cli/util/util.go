// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package util

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/harness/gitness/cli/session"
	"github.com/harness/gitness/client"

	"github.com/adrg/xdg"
)

const (
	OwnerReadWrite = 0600
)

// Client returns a client that is configured from the default session file.
func Client() (*client.HTTPClient, error) {
	session, err := LoadSession()
	if err != nil {
		return nil, err
	}

	client := client.NewToken(session.URI, session.AccessToken)
	if os.Getenv("DEBUG") == "true" {
		client.SetDebug(true)
	}
	return client, nil
}

// LoadSession loads an existing session from the default file.
func LoadSession() (*session.Session, error) {
	path, err := Config()
	if err != nil {
		return nil, err
	}

	return LoadSessionFromPath(path)
}

// LoadSessionFromPath loads an existing session from a file.
func LoadSessionFromPath(path string) (*session.Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session from file: %w", err)
	}
	session := new(session.Session)
	if err = json.Unmarshal(data, session); err != nil {
		return nil, fmt.Errorf("failed to deserialize session: %w", err)
	}

	if time.Now().Unix() > session.ExpiresAt {
		return nil, errors.New("token is expired, please login")
	}

	return session, nil
}

// StoreSession stores an existing session to the default file.
func StoreSession(session *session.Session) error {
	path, err := Config()
	if err != nil {
		return err
	}

	return StoreSessionAtPath(path, session)
}

// StoreSessionAtPath writes a session to a file.
func StoreSessionAtPath(path string, session *session.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to serialize session: %w", err)
	}

	err = os.WriteFile(path, data, OwnerReadWrite)
	if err != nil {
		return fmt.Errorf("failed to write session to file: %w", err)
	}

	return nil
}

// Config returns the configuration file path.
func Config() (string, error) {
	return xdg.ConfigFile(
		filepath.Join("app", "config.json"),
	)
}

// Registration returns the username, name, email and password from stdin.
func Registration() (string, string, string, string) {
	return Username(), Name(), Email(), Password()
}

// Credentials returns the username and password from stdin.
func Credentials() (string, string) {
	return Username(), Password()
}

// Username returns the username from stdin.
func Username() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	return strings.TrimSpace(username)
}

// Name returns the name from stdin.
func Name() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Name: ")
	name, _ := reader.ReadString('\n')

	return strings.TrimSpace(name)
}

// Email returns the email from stdin.
func Email() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Email: ")
	email, _ := reader.ReadString('\n')

	return strings.TrimSpace(email)
}

// Password returns the password from stdin.
func Password() string {
	fmt.Print("Enter Password: ")
	passwordb, _ := term.ReadPassword(syscall.Stdin)
	password := string(passwordb)

	return strings.TrimSpace(password)
}
