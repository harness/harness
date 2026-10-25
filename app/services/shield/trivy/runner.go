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

package trivy

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
	DefaultBinary         = "trivy"
	DefaultTimeout        = 10 * time.Minute
	DefaultMaxOutputBytes = 64 << 20
)

// ErrBinaryMissing is returned when the trivy binary cannot be found.
var ErrBinaryMissing = errors.New("trivy binary not found")

// ErrOutputTooLarge is returned when the trivy output exceeds MaxOutputBytes.
var ErrOutputTooLarge = errors.New("trivy output exceeds limit")

// Options configure a scan.
type Options struct {
	// Binary is the trivy executable (name or path). Default "trivy".
	Binary string
	// Timeout bounds the scan. Default 10 minutes.
	Timeout time.Duration
	// MaxOutputBytes bounds the SARIF size held in memory. Default 64 MiB.
	MaxOutputBytes int64
	// SkipDBUpdate adds --skip-db-update (image scans); the vulnerability DB must already be
	// present in the cache. Use it in air-gapped installations.
	SkipDBUpdate bool
	// SkipCheckUpdate adds --skip-check-update (config scans); the bundled checks are used
	// instead of downloading the latest ones.
	SkipCheckUpdate bool
	// CacheDir adds --cache-dir to point trivy at a pre-populated cache.
	CacheDir string
}

// Runner runs trivy and returns the SARIF output.
type Runner interface {
	ScanImage(ctx context.Context, ref string, opts Options) (io.ReadCloser, error)
	ScanConfig(ctx context.Context, dir string, opts Options) (io.ReadCloser, error)
}

// ExecRunner runs trivy as a child process.
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
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.MaxOutputBytes <= 0 {
		o.MaxOutputBytes = DefaultMaxOutputBytes
	}
	return o
}

func (o Options) cacheArgs() []string {
	if o.CacheDir == "" {
		return nil
	}
	return []string{"--cache-dir", o.CacheDir}
}

// ImageArgs returns the arguments of an image scan:
// image --format sarif --quiet --scanners vuln [--skip-db-update] [--cache-dir D] -- <ref>.
func ImageArgs(ref string, opts Options) []string {
	args := []string{"image", "--format", "sarif", "--quiet", "--scanners", "vuln"}
	if opts.SkipDBUpdate {
		args = append(args, "--skip-db-update")
	}
	args = append(args, opts.cacheArgs()...)
	return append(args, "--", ref)
}

// ConfigArgs returns the arguments of an IaC misconfiguration scan:
// config --format sarif --quiet [--skip-check-update] [--cache-dir D] -- <dir>.
func ConfigArgs(dir string, opts Options) []string {
	args := []string{"config", "--format", "sarif", "--quiet"}
	if opts.SkipCheckUpdate {
		args = append(args, "--skip-check-update")
	}
	args = append(args, opts.cacheArgs()...)
	return append(args, "--", dir)
}

func (r *ExecRunner) command(ctx context.Context, opts Options, args []string) (*exec.Cmd, error) {
	look := r.LookPath
	if look == nil {
		look = exec.LookPath
	}
	bin, err := look(opts.Binary)
	if err != nil {
		return nil, fmt.Errorf("%w: %q (install trivy or set the binary path): %w",
			ErrBinaryMissing, opts.Binary, err)
	}
	return exec.CommandContext(ctx, bin, args...), nil //nolint:gosec // binary is operator-configured
}

// ImageCommand builds the image scan command without running it.
func (r *ExecRunner) ImageCommand(ctx context.Context, ref string, opts Options) (*exec.Cmd, error) {
	opts = opts.withDefaults()
	return r.command(ctx, opts, ImageArgs(ref, opts))
}

// ConfigCommand builds the config scan command without running it.
func (r *ExecRunner) ConfigCommand(ctx context.Context, dir string, opts Options) (*exec.Cmd, error) {
	opts = opts.withDefaults()
	cmd, err := r.command(ctx, opts, ConfigArgs(dir, opts))
	if err != nil {
		return nil, err
	}
	cmd.Dir = dir
	return cmd, nil
}

// ScanImage scans a container image reference and returns the SARIF document; the caller must
// close it.
func (r *ExecRunner) ScanImage(ctx context.Context, ref string, opts Options) (io.ReadCloser, error) {
	opts = opts.withDefaults()
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	cmd, err := r.ImageCommand(ctx, ref, opts)
	if err != nil {
		return nil, err
	}
	return run(ctx, cmd, opts)
}

// ScanConfig scans the IaC files in dir and returns the SARIF document; the caller must close it.
func (r *ExecRunner) ScanConfig(ctx context.Context, dir string, opts Options) (io.ReadCloser, error) {
	opts = opts.withDefaults()
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	cmd, err := r.ConfigCommand(ctx, dir, opts)
	if err != nil {
		return nil, err
	}
	return run(ctx, cmd, opts)
}

func run(ctx context.Context, cmd *exec.Cmd, opts Options) (io.ReadCloser, error) {
	out := &limitedBuffer{limit: opts.MaxOutputBytes}
	stderr := &limitedBuffer{limit: 8 << 10, truncate: true}
	cmd.Stdout = out
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		switch {
		case out.exceeded:
			return nil, ErrOutputTooLarge
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return nil, fmt.Errorf("trivy scan timed out after %s: %w", opts.Timeout, ctx.Err())
		case ctx.Err() != nil:
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("trivy scan failed: %w: %s", err, bytes.TrimSpace(stderr.buf.Bytes()))
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
