// Copyright 2023 Harness, Inc.
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

package command

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/errors"

	"github.com/rs/zerolog/log"
)

func TestCreateBareRepository(t *testing.T) {
	cmd := New("init", WithFlag("--bare"), WithArg("samplerepo"))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := cmd.Run(ctx)
	defer os.RemoveAll("samplerepo")
	if err != nil {
		t.Errorf("expected: %v error, got: %v", nil, err)
		return
	}

	cmd = New("rev-parse", WithFlag("--is-bare-repository"))
	output := &bytes.Buffer{}
	err = cmd.Run(context.Background(), WithDir("samplerepo"), WithStdout(output))
	if err != nil {
		t.Errorf("expected: %v error, got: %v", nil, err)
		return
	}
	got := strings.TrimSpace(output.String())
	exp := "true"
	if got != exp {
		t.Errorf("expected value: %s, got: %s", exp, got)
		return
	}
}

func TestArchiveValidatePositionalArgs(t *testing.T) {
	b := descriptions["archive"]

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "plain paths",
			args:    []string{"README.md", "docs"},
			wantErr: false,
		},
		{
			name:    "leading compression level then paths",
			args:    []string{"-9", "README.md", "docs"},
			wantErr: false,
		},
		{
			name:    "compression level only",
			args:    []string{"-9"},
			wantErr: false,
		},
		{
			name:    "flag smuggled after compression level is rejected",
			args:    []string{"-9", "--add-file=/etc/passwd"},
			wantErr: true,
		},
		{
			name:    "output flag smuggled after compression level is rejected",
			args:    []string{"-9", "--output=/tmp/pwned"},
			wantErr: true,
		},
		{
			name:    "remote flag smuggled after compression level is rejected",
			args:    []string{"-9", "--remote=git://attacker/x"},
			wantErr: true,
		},
		{
			name:    "flag as first arg is rejected",
			args:    []string{"--add-file=/etc/passwd"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := b.validatePositionalArgs(tt.args)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for args %v, got nil", tt.args)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error for args %v, got %v", tt.args, err)
			}
		})
	}
}

func TestCommandContextTimeout(t *testing.T) {
	cmd := New("init", WithFlag("--bare"), WithArg("samplerepo"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := cmd.Run(ctx)
	defer os.RemoveAll("samplerepo")
	if err != nil {
		t.Errorf("expected: %v error, got: %v", nil, err)
	}

	inbuff := &bytes.Buffer{}
	inbuff.WriteString("some content")
	outbuffer := &bytes.Buffer{}

	cmd = New("hash-object", WithFlag("--stdin"))
	err = cmd.Run(ctx,
		WithDir("./samplerepo"),
		WithStdin(inbuff),
		WithStdout(outbuffer),
	)
	if err != nil {
		t.Errorf("hashing object failed: %v", err)
		return
	}

	log.Info().Msgf("outbuffer %s", outbuffer.String())

	cmd = New("cat-file", WithFlag("--batch"))

	pr, pw := io.Pipe()
	defer pr.Close()

	outbuffer.Reset()

	go func() {
		defer pw.Close()
		for range 3 {
			_, _ = pw.Write(outbuffer.Bytes())
			time.Sleep(1 * time.Second)
		}
	}()

	runCtx, runCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer runCancel()

	err = cmd.Run(runCtx,
		WithDir("./samplerepo"),
		WithStdin(pr),
		WithStdout(outbuffer),
	)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected: %v error, got: %v", context.DeadlineExceeded, err)
	}
}
