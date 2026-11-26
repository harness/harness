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

// TestMakeArgs_LiteralPathspecs asserts that --literal-pathspecs is passed only when requested,
// and that it comes before the sub command, otherwise git rejects it.
//
// Regression guard for CODE-6237: without the flag git reads a path inside a directory
// named ":" as the magic "relative to the repository root" prefix, which made committing
// such a file fail.
func TestMakeArgs_LiteralPathspecs(t *testing.T) {
	t.Run("requested", func(t *testing.T) {
		cmd := New("ls-tree",
			WithLiteralPathspecs(),
			WithFlag("-z"),
			WithArg("main"),
			WithPostSepArg(":/f.txt"),
		)

		args, err := cmd.makeArgs()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(args) < 2 || args[0] != "--literal-pathspecs" || args[1] != "ls-tree" {
			t.Fatalf("expected --literal-pathspecs right before the sub command, got: %v", args)
		}
	})

	t.Run("not requested", func(t *testing.T) {
		args, err := New("ls-tree", WithArg("main")).makeArgs()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if args[0] != "ls-tree" {
			t.Fatalf("expected no global flags by default, got: %v", args)
		}
	})
}

// TestClone_KeepsPostSepArgs asserts that a cloned command keeps its post-separator arguments,
// and does not fill them with its flags.
func TestClone_KeepsPostSepArgs(t *testing.T) {
	cmd := New("ls-tree", WithFlag("-z"), WithArg("main"), WithPostSepArg(":/f.txt"))

	clone := cmd.Clone()
	if len(clone.PostSepArgs) != 1 || clone.PostSepArgs[0] != ":/f.txt" {
		t.Fatalf("expected the post separator args to be kept, got: %v", clone.PostSepArgs)
	}
}

// TestClone_KeepsGlobals asserts that a cloned command keeps its global flags, so that
// --literal-pathspecs survives for the callers that clone a command before running it.
func TestClone_KeepsGlobals(t *testing.T) {
	cmd := New("diff", WithLiteralPathspecs(), WithFlag("-M"))

	args, err := cmd.Clone().makeArgs()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(args) < 2 || args[0] != "--literal-pathspecs" || args[1] != "diff" {
		t.Fatalf("expected --literal-pathspecs to survive the clone, got: %v", args)
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

func TestCommandContextCanceled(t *testing.T) {
	repoDir := t.TempDir()
	if err := New("init", WithFlag("--bare")).Run(context.Background(), WithDir(repoDir)); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// cat-file --batch keeps running for as long as stdin stays open. The pipe must still close
	// eventually, because cmd.Wait also waits for the goroutine copying stdin.
	pr, pw := io.Pipe()
	defer pr.Close()
	time.AfterFunc(time.Second, func() { _ = pw.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)

	err := New("cat-file", WithFlag("--batch")).Run(ctx,
		WithDir(repoDir),
		WithStdin(pr),
		WithStdout(io.Discard),
	)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected: %v error, got: %v", context.Canceled, err)
	}
}
