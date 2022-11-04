// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package textui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

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
