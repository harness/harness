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
		for i := 0; i < 3; i++ {
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
