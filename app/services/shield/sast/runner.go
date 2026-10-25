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

package sast

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// Defaults of the exec runner.
const (
	DefaultBinary         = "opengrep"
	DefaultConfig         = "p/default"
	DefaultTimeout        = 10 * time.Minute
	DefaultMaxOutputBytes = 64 << 20
)

// ErrBinaryMissing is returned when the scanner binary cannot be found.
var ErrBinaryMissing = errors.New("sast scanner binary not found")

// ErrOutputTooLarge is returned when the scanner output exceeds MaxOutputBytes.
var ErrOutputTooLarge = errors.New("sast scanner output exceeds limit")

// Options configure a scan.
type Options struct {
	// Binary is the scanner executable ("opengrep" or "semgrep", name or path). Default "opengrep".
	Binary string
	// Config is a ruleset: a local rules file/directory (works offline) or a registry ruleset
	// such as the default "p/default", which is downloaded from the registry and therefore
	// needs network access. Use a local rules directory in air-gapped installations.
	Config string
	// Timeout bounds the scan. Default 10 minutes.
	Timeout time.Duration
	// MaxOutputBytes bounds the SARIF size held in memory. Default 64 MiB.
	MaxOutputBytes int64
}

// Runner runs a SAST scan over a directory and returns the SARIF output.
type Runner interface {
	Run(ctx context.Context, dir string, opts Options) (io.ReadCloser, error)
}

// ExecRunner runs the scanner as a child process.
type ExecRunner struct {
	// LookPath resolves the binary; defaults to exec.LookPath (overridable for tests).
	LookPath func(string) (string, error)
}

// NewExecRunner returns an ExecRunner.
func NewExecRunner() *ExecRunner { return &ExecRunner{LookPath: exec.LookPath} }

func (o Options) withDefaults() Options {
	if o.Binary == "" {
		o.Binary = DefaultBinary
	}
	if o.Config == "" {
		o.Config = DefaultConfig
	}
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.MaxOutputBytes <= 0 {
		o.MaxOutputBytes = DefaultMaxOutputBytes
	}
	return o
}

// Args returns the scanner arguments: scan --sarif --config <config> --metrics=off --quiet <dir>.
func Args(dir string, opts Options) []string {
	opts = opts.withDefaults()
	return []string{"scan", "--sarif", "--config", opts.Config, "--metrics=off", "--quiet", dir}
}

// Command builds the scanner command without running it.
func (r *ExecRunner) Command(ctx context.Context, dir string, opts Options) (*exec.Cmd, error) {
	opts = opts.withDefaults()
	look := r.LookPath
	if look == nil {
		look = exec.LookPath
	}
	bin, err := look(opts.Binary)
	if err != nil {
		return nil, fmt.Errorf("%w: %q (install opengrep or semgrep, or set the binary path): %w",
			ErrBinaryMissing, opts.Binary, err)
	}
	cmd := exec.CommandContext(ctx, bin, Args(dir, opts)...) //nolint:gosec // binary is operator-configured
	cmd.Dir = dir
	return cmd, nil
}

// Run scans dir and returns the SARIF document; the caller must close it.
func (r *ExecRunner) Run(ctx context.Context, dir string, opts Options) (io.ReadCloser, error) {
	opts = opts.withDefaults()
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	cmd, err := r.Command(ctx, dir, opts)
	if err != nil {
		return nil, err
	}
	out := &limitedBuffer{limit: opts.MaxOutputBytes}
	stderr := &limitedBuffer{limit: 8 << 10, truncate: true}
	cmd.Stdout = out
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		switch {
		case out.exceeded:
			return nil, ErrOutputTooLarge
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return nil, fmt.Errorf("sast scan timed out after %s: %w", opts.Timeout, ctx.Err())
		case ctx.Err() != nil:
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("sast scan failed: %w: %s", err, bytes.TrimSpace(stderr.buf.Bytes()))
	}
	return io.NopCloser(bytes.NewReader(out.buf.Bytes())), nil
}

type limitedBuffer struct {
	buf      bytes.Buffer
	limit    int64
	truncate bool // silently drop overflow instead of failing
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - int64(b.buf.Len())
	if int64(len(p)) > remaining {
		if b.truncate {
			b.buf.Write(p[:max(remaining, 0)])
			return len(p), nil
		}
		b.exceeded = true
		return 0, ErrOutputTooLarge
	}
	return b.buf.Write(p)
}
