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

package hook

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	gtxSHAZero = "0000000000000000000000000000000000000000"
	gtxSHAOne  = "1111111111111111111111111111111111111111"
	gtxSHATwo  = "2222222222222222222222222222222222222222"
)

// gtxFakeClient records the calls and returns the configured response.
type gtxFakeClient struct {
	out Output
	err error

	preInputs    []PreReceiveInput
	updateInputs []UpdateInput
	postInputs   []PostReceiveInput
}

func (c *gtxFakeClient) PreReceive(_ context.Context, in PreReceiveInput) (Output, error) {
	c.preInputs = append(c.preInputs, in)
	return c.out, c.err
}

func (c *gtxFakeClient) Update(_ context.Context, in UpdateInput) (Output, error) {
	c.updateInputs = append(c.updateInputs, in)
	return c.out, c.err
}

func (c *gtxFakeClient) PostReceive(_ context.Context, in PostReceiveInput) (Output, error) {
	c.postInputs = append(c.postInputs, in)
	return c.out, c.err
}

// gtxSetStdin replaces os.Stdin with a file holding the content until the test ends.
func gtxSetStdin(t *testing.T, content string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "stdin")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	f, err := os.Open(path)
	require.NoError(t, err)

	orig := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = orig
		_ = f.Close()
	})
}

// gtxCaptureStdout returns everything the function writes to os.Stdout.
func gtxCaptureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	orig := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()

	func() {
		defer func() {
			os.Stdout = orig
			_ = w.Close()
		}()
		fn()
	}()

	out := <-done
	_ = r.Close()

	return out
}

func gtxStdinLine(old, newSHA, ref string) string {
	return old + " " + newSHA + " " + ref + "\n"
}

func TestGtxSanitizeArgsForGit(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		want    []string
		wantOK  bool
	}{
		{
			name: "pre-receive", command: CommandNamePreReceive, args: nil,
			want: []string{ParamPreReceive}, wantOK: true,
		},
		{
			name: "update", command: CommandNameUpdate, args: []string{"refs/heads/main", "a", "b"},
			want: []string{ParamUpdate, "refs/heads/main", "a", "b"}, wantOK: true,
		},
		{
			name: "post-receive", command: CommandNamePostReceive, args: []string{"x"},
			want: []string{ParamPostReceive, "x"}, wantOK: true,
		},
		{name: "other command", command: "gitness", args: []string{"server"}, want: []string{"server"}},
		{name: "empty command", command: "", args: nil, want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := SanitizeArgsForGit(test.command, test.args)
			assert.Equal(t, test.wantOK, ok)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGtxCLICore_PreReceive(t *testing.T) {
	ctx := context.Background()

	t.Run("create and update need the object dir", func(t *testing.T) {
		t.Setenv(command.GitObjectDir, "/gtx/quarantine")
		gtxSetStdin(t, gtxStdinLine(gtxSHAZero, gtxSHAOne, "refs/heads/new")+
			gtxStdinLine(gtxSHAOne, gtxSHATwo, "refs/heads/main"))

		client := &gtxFakeClient{}
		require.NoError(t, NewCLICore(client, time.Second).PreReceive(ctx))

		require.Len(t, client.preInputs, 1)
		in := client.preInputs[0]
		assert.Equal(t, []string{"/gtx/quarantine"}, in.Environment.AlternateObjectDirs)
		assert.Equal(t, []ReferenceUpdate{
			{Ref: "refs/heads/new", Old: sha.Nil, New: sha.Must(gtxSHAOne)},
			{Ref: "refs/heads/main", Old: sha.Must(gtxSHAOne), New: sha.Must(gtxSHATwo)},
		}, in.RefUpdates)
	})

	t.Run("delete only doesn't need the object dir", func(t *testing.T) {
		t.Setenv(command.GitObjectDir, "")
		require.NoError(t, os.Unsetenv(command.GitObjectDir))
		gtxSetStdin(t, gtxStdinLine(gtxSHAOne, gtxSHAZero, "refs/heads/old"))

		client := &gtxFakeClient{}
		require.NoError(t, NewCLICore(client, time.Second).PreReceive(ctx))

		require.Len(t, client.preInputs, 1)
		assert.Empty(t, client.preInputs[0].Environment.AlternateObjectDirs)
		assert.Len(t, client.preInputs[0].RefUpdates, 1)
	})

	t.Run("no updates", func(t *testing.T) {
		gtxSetStdin(t, "")

		client := &gtxFakeClient{}
		require.NoError(t, NewCLICore(client, time.Second).PreReceive(ctx))

		require.Len(t, client.preInputs, 1)
		assert.Empty(t, client.preInputs[0].RefUpdates)
	})

	t.Run("missing object dir", func(t *testing.T) {
		require.NoError(t, os.Unsetenv(command.GitObjectDir))
		gtxSetStdin(t, gtxStdinLine(gtxSHAZero, gtxSHAOne, "refs/heads/new"))

		client := &gtxFakeClient{}
		err := NewCLICore(client, time.Second).PreReceive(ctx)
		require.ErrorContains(t, err, command.GitObjectDir)
		assert.Empty(t, client.preInputs)
	})

	t.Run("empty object dir", func(t *testing.T) {
		t.Setenv(command.GitObjectDir, "")
		gtxSetStdin(t, gtxStdinLine(gtxSHAZero, gtxSHAOne, "refs/heads/new"))

		err := NewCLICore(&gtxFakeClient{}, time.Second).PreReceive(ctx)
		require.ErrorContains(t, err, "empty")
	})

	t.Run("malformed stdin", func(t *testing.T) {
		gtxSetStdin(t, "two fields\n")

		client := &gtxFakeClient{}
		err := NewCLICore(client, time.Second).PreReceive(ctx)
		require.ErrorContains(t, err, "invalid data format")
		assert.Empty(t, client.preInputs)
	})

	t.Run("server error", func(t *testing.T) {
		gtxSetStdin(t, "")

		err := NewCLICore(&gtxFakeClient{err: errors.New("gtx unavailable")}, time.Second).PreReceive(ctx)
		require.ErrorContains(t, err, "gtx unavailable")
		require.ErrorContains(t, err, "error occurred when calling the server")
	})

	t.Run("server rejects the push", func(t *testing.T) {
		gtxSetStdin(t, "")

		msg := "branch is protected"
		client := &gtxFakeClient{out: Output{Error: &msg}}

		err := NewCLICore(client, time.Second).PreReceive(ctx)
		require.EqualError(t, err, msg)
	})
}

func TestGtxCLICore_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("update with object dir", func(t *testing.T) {
		t.Setenv(command.GitObjectDir, "/gtx/quarantine")

		client := &gtxFakeClient{}
		require.NoError(t, NewCLICore(client, time.Second).Update(ctx, "refs/heads/main", gtxSHAOne, gtxSHATwo))

		require.Len(t, client.updateInputs, 1)
		assert.Equal(t, UpdateInput{
			Environment: Environment{AlternateObjectDirs: []string{"/gtx/quarantine"}},
			RefUpdate:   ReferenceUpdate{Ref: "refs/heads/main", Old: sha.Must(gtxSHAOne), New: sha.Must(gtxSHATwo)},
		}, client.updateInputs[0])
	})

	t.Run("delete doesn't need object dir", func(t *testing.T) {
		require.NoError(t, os.Unsetenv(command.GitObjectDir))

		client := &gtxFakeClient{}
		require.NoError(t, NewCLICore(client, time.Second).Update(ctx, "refs/heads/old", gtxSHAOne, gtxSHAZero))

		require.Len(t, client.updateInputs, 1)
		assert.Empty(t, client.updateInputs[0].Environment.AlternateObjectDirs)
	})

	t.Run("missing object dir", func(t *testing.T) {
		require.NoError(t, os.Unsetenv(command.GitObjectDir))

		err := NewCLICore(&gtxFakeClient{}, time.Second).Update(ctx, "refs/heads/main", gtxSHAOne, gtxSHATwo)
		require.ErrorContains(t, err, command.GitObjectDir)
	})

	t.Run("server error", func(t *testing.T) {
		require.NoError(t, os.Unsetenv(command.GitObjectDir))

		client := &gtxFakeClient{err: errors.New("gtx unavailable")}
		err := NewCLICore(client, time.Second).Update(ctx, "refs/heads/old", gtxSHAOne, gtxSHAZero)
		require.ErrorContains(t, err, "gtx unavailable")
	})
}

func TestGtxCLICore_PostReceive(t *testing.T) {
	ctx := context.Background()

	t.Run("references and no alternate dirs", func(t *testing.T) {
		t.Setenv(command.GitObjectDir, "/gtx/quarantine")
		gtxSetStdin(t, gtxStdinLine(gtxSHAOne, gtxSHATwo, "refs/heads/main"))

		client := &gtxFakeClient{}
		require.NoError(t, NewCLICore(client, time.Second).PostReceive(ctx))

		require.Len(t, client.postInputs, 1)
		assert.Empty(t, client.postInputs[0].Environment.AlternateObjectDirs)
		assert.Equal(t,
			[]ReferenceUpdate{{Ref: "refs/heads/main", Old: sha.Must(gtxSHAOne), New: sha.Must(gtxSHATwo)}},
			client.postInputs[0].RefUpdates)
	})

	t.Run("malformed stdin", func(t *testing.T) {
		gtxSetStdin(t, "a b c d\n")

		require.Error(t, NewCLICore(&gtxFakeClient{}, time.Second).PostReceive(ctx))
	})

	t.Run("server error", func(t *testing.T) {
		gtxSetStdin(t, "")

		err := NewCLICore(&gtxFakeClient{err: errors.New("gtx unavailable")}, time.Second).PostReceive(ctx)
		require.ErrorContains(t, err, "gtx unavailable")
	})
}

func TestGtxHandleServerHookOutput(t *testing.T) {
	errMsg := "rejected"

	tests := []struct {
		name       string
		out        Output
		err        error
		wantStdout string
		wantErr    string
	}{
		{name: "nothing", out: Output{}},
		{name: "only empty messages", out: Output{Messages: []string{"", ""}}},
		{
			name:       "messages are framed and trimmed",
			out:        Output{Messages: []string{"", "first", "", "second", ""}},
			wantStdout: "\nfirst\n\nsecond\n\n",
		},
		{
			name:       "messages are printed before the error",
			out:        Output{Messages: []string{"hint"}, Error: &errMsg},
			wantStdout: "\nhint\n\n",
			wantErr:    "rejected",
		},
		{name: "error only", out: Output{Error: &errMsg}, wantErr: "rejected"},
		{name: "call error wins", err: errors.New("boom"), wantErr: "boom"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			stdout := gtxCaptureStdout(t, func() { err = handleServerHookOutput(test.out, test.err) })

			assert.Equal(t, test.wantStdout, stdout)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestGtxRegisterAll(t *testing.T) {
	newApp := func() *kingpin.Application {
		app := kingpin.New("gtx", "")
		app.Terminate(nil)
		return app
	}

	t.Run("pre-receive", func(t *testing.T) {
		gtxSetStdin(t, "")
		client := &gtxFakeClient{}
		app := newApp()
		RegisterAll(app, func() (*CLICore, error) { return NewCLICore(client, time.Second), nil })

		_, err := app.Parse([]string{ParamPreReceive})
		require.NoError(t, err)
		assert.Len(t, client.preInputs, 1)
	})

	t.Run("update", func(t *testing.T) {
		require.NoError(t, os.Unsetenv(command.GitObjectDir))
		client := &gtxFakeClient{}
		app := newApp()
		RegisterAll(app, func() (*CLICore, error) { return NewCLICore(client, time.Second), nil })

		_, err := app.Parse([]string{ParamUpdate, "refs/heads/old", gtxSHAOne, gtxSHAZero})
		require.NoError(t, err)

		require.Len(t, client.updateInputs, 1)
		assert.Equal(t, "refs/heads/old", client.updateInputs[0].RefUpdate.Ref)
	})

	t.Run("update requires arguments", func(t *testing.T) {
		app := newApp()
		RegisterUpdate(app, func() (*CLICore, error) { return nil, errors.New("must not be called") })

		_, err := app.Parse([]string{ParamUpdate, "refs/heads/old"})
		require.Error(t, err)
	})

	t.Run("post-receive", func(t *testing.T) {
		gtxSetStdin(t, gtxStdinLine(gtxSHAOne, gtxSHATwo, "refs/heads/main"))
		client := &gtxFakeClient{}
		app := newApp()
		RegisterPostReceive(app, func() (*CLICore, error) { return NewCLICore(client, time.Second), nil })

		_, err := app.Parse([]string{ParamPostReceive})
		require.NoError(t, err)
		assert.Len(t, client.postInputs, 1)
	})

	t.Run("disabled hooks are skipped", func(t *testing.T) {
		client := &gtxFakeClient{}
		app := newApp()
		RegisterAll(app, func() (*CLICore, error) { return nil, ErrDisabled })

		for _, args := range [][]string{
			{ParamPreReceive},
			{ParamUpdate, "refs/heads/main", gtxSHAOne, gtxSHATwo},
			{ParamPostReceive},
		} {
			_, err := app.Parse(args)
			require.NoError(t, err, args)
		}
		assert.Empty(t, client.preInputs)
	})

	t.Run("loader failure", func(t *testing.T) {
		loadErr := errors.New("gtx load failure")
		app := newApp()
		RegisterAll(app, func() (*CLICore, error) { return nil, loadErr })

		for _, args := range [][]string{
			{ParamPreReceive},
			{ParamUpdate, "refs/heads/main", gtxSHAOne, gtxSHATwo},
			{ParamPostReceive},
		} {
			_, err := app.Parse(args)
			require.ErrorIs(t, err, loadErr, args)
		}
	})
}

type gtxPayload struct {
	Name  string
	Items []int
	Refs  []ReferenceUpdate
}

func TestGtxPayloadEnvironment(t *testing.T) {
	want := gtxPayload{
		Name:  "gtx",
		Items: []int{1, 2, 3},
		Refs:  []ReferenceUpdate{{Ref: "refs/heads/main", Old: sha.Must(gtxSHAOne), New: sha.Must(gtxSHATwo)}},
	}

	envVars, err := GenerateEnvironmentVariables(want)
	require.NoError(t, err)
	require.Len(t, envVars, 1)
	require.Contains(t, envVars, envNamePayload)

	t.Run("from map", func(t *testing.T) {
		got, err := LoadPayloadFromMap[gtxPayload](envVars)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("from environment", func(t *testing.T) {
		t.Setenv(envNamePayload, envVars[envNamePayload])

		got, err := LoadPayloadFromEnvironment[gtxPayload]()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("map without payload", func(t *testing.T) {
		_, err := LoadPayloadFromMap[gtxPayload](map[string]string{})
		require.ErrorContains(t, err, envNamePayload)
	})

	t.Run("map with invalid base64", func(t *testing.T) {
		_, err := LoadPayloadFromMap[gtxPayload](map[string]string{envNamePayload: "***"})
		require.ErrorContains(t, err, "base64")
	})

	t.Run("map with invalid gob", func(t *testing.T) {
		_, err := LoadPayloadFromMap[gtxPayload](map[string]string{envNamePayload: "Z2FyYmFnZQ=="})
		require.ErrorContains(t, err, "deserialize")
	})

	t.Run("environment without payload", func(t *testing.T) {
		require.NoError(t, os.Unsetenv(envNamePayload))

		_, err := LoadPayloadFromEnvironment[gtxPayload]()
		require.ErrorContains(t, err, "failed to load payload from environment")
	})

	t.Run("environment with empty payload", func(t *testing.T) {
		t.Setenv(envNamePayload, "")

		_, err := LoadPayloadFromEnvironment[gtxPayload]()
		require.ErrorContains(t, err, "empty")
	})

	t.Run("payload that can't be encoded", func(t *testing.T) {
		_, err := GenerateEnvironmentVariables(make(chan int))
		require.ErrorContains(t, err, "failed to encode payload")
	})
}

func TestGtxEnvVarsToMap(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		want    map[string]string
		wantErr bool
	}{
		{name: "nil", in: nil, want: map[string]string{}},
		{
			name: "entries", in: []string{"A=1", "B=two words", " C =3", "D="},
			want: map[string]string{"A": "1", "B": "two words", "C": "3", "D": ""},
		},
		{name: "value with equal signs", in: []string{"A=b=c"}, want: map[string]string{"A": "b=c"}},
		{name: "later entries win", in: []string{"A=1", "A=2"}, want: map[string]string{"A": "2"}},
		{name: "missing equal sign", in: []string{"A=1", "broken"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := EnvVarsToMap(test.in)
			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGtxNoopClient(t *testing.T) {
	ctx := context.Background()
	messages := []string{"hello", "world"}
	client := NewNoopClient(messages)

	pre, err := client.PreReceive(ctx, PreReceiveInput{})
	require.NoError(t, err)
	assert.Equal(t, Output{Messages: messages}, pre)

	update, err := client.Update(ctx, UpdateInput{})
	require.NoError(t, err)
	assert.Equal(t, Output{Messages: messages}, update)

	post, err := client.PostReceive(ctx, PostReceiveInput{})
	require.NoError(t, err)
	assert.Equal(t, Output{Messages: messages}, post)

	empty, err := NewNoopClient(nil).PreReceive(ctx, PreReceiveInput{})
	require.NoError(t, err)
	assert.Empty(t, empty.Messages)
	assert.Nil(t, empty.Error)
}

func TestGtxRefUpdaterStateString(t *testing.T) {
	tests := map[refUpdaterState]string{
		stateInit:   "INIT",
		statePre:    "PRE",
		stateUpdate: "UPDATE",
		statePost:   "POST",
		stateDone:   "DONE",
		255:         "INVALID",
	}

	for state, want := range tests {
		assert.Equal(t, want, state.String())
	}
}

func TestGtxGetUpdatedReferencesFromStdIn_NoTrailingNewline(t *testing.T) {
	// git always terminates the lines, so a trailing partial line isn't a valid record.
	gtxSetStdin(t, gtxStdinLine(gtxSHAOne, gtxSHATwo, "refs/heads/a")+"partial")

	got, err := getUpdatedReferencesFromStdIn()
	require.NoError(t, err)
	require.NotEmpty(t, got)
	assert.Equal(t, "refs/heads/a", got[0].Ref)
	assert.True(t, strings.HasPrefix(got[0].Old.String(), "1111"))
}
