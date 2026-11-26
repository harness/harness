// Copyright 2026 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package textui

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

// infWithStdin replaces os.Stdin with a pipe that already contains the given content.
func infWithStdin(t *testing.T, content string) {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = old
		_ = r.Close()
	})
}

func TestInfLineInputs(t *testing.T) {
	tests := []struct {
		name string
		fn   func() string
	}{
		{"user id", UserID},
		{"login identifier", LoginIdentifier},
		{"display name", DisplayName},
		{"email", Email},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infWithStdin(t, "  some value \nignored second line\n")
			require.Equal(t, "some value", tt.fn(), "input is trimmed and limited to the first line")
		})
	}
}

func TestInfLineInputs_EmptyStdin(t *testing.T) {
	infWithStdin(t, "")
	require.Empty(t, UserID())
	infWithStdin(t, "\n")
	require.Empty(t, Email())
}

func TestInfPassword_WithoutTerminal(t *testing.T) {
	infSkipIfTerminal(t)
	// stdin is not a terminal in tests, so no password can be read.
	infWithStdin(t, "")
	require.Empty(t, Password())
}

func TestInfCredentialsAndRegistrationWithoutTerminal(t *testing.T) {
	infSkipIfTerminal(t)
	infWithStdin(t, "")
	login, password := Credentials()
	require.Empty(t, login)
	require.Empty(t, password)

	infWithStdin(t, "")
	uid, name, email, pass := Registration()
	require.Empty(t, uid)
	require.Empty(t, name)
	require.Empty(t, email)
	require.Empty(t, pass)
}

// infSkipIfTerminal skips tests that would block on reading a password from an interactive terminal.
func infSkipIfTerminal(t *testing.T) {
	t.Helper()
	if term.IsTerminal(syscall.Stdin) {
		t.Skip("stdin is an interactive terminal")
	}
}
