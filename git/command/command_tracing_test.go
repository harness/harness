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
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// The package tracer is bound to the first global tracer provider, so it is only set once per test binary.
var (
	recorder     = tracetest.NewSpanRecorder()
	setProvider  sync.Once
	testProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
)

func TestRunTracing(t *testing.T) {
	setProvider.Do(func() { otel.SetTracerProvider(testProvider) })

	before := len(recorder.Ended())
	require.NoError(t, New("version").Run(context.Background()))
	assert.Len(t, recorder.Ended(), before, "a command outside of a traced operation must not create a span")

	ctx, parent := testProvider.Tracer("test").Start(context.Background(), "parent")

	require.NoError(t, New("version").Run(ctx))

	// git exits with 1 if the key doesn't exist, which is a regular result and not a failure.
	configFile := filepath.Join(t.TempDir(), "config")
	err := New("config", WithFlag("--file", configFile), WithFlag("--get"), WithArg("missing.key")).Run(ctx)
	require.Error(t, err)

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Nanosecond)
	defer cancel()
	<-timeoutCtx.Done()
	_ = New("version").Run(timeoutCtx)

	// Callers cancel the context to stop long-running processes, which isn't a failure.
	canceledCtx, cancelRun := context.WithCancel(ctx)
	cancelRun()
	_ = New("version").Run(canceledCtx)

	repoDir := t.TempDir()
	require.NoError(t, exec.CommandContext(t.Context(), GitExecutable, "init", "-q", repoDir).Run())

	stopCtx, stop := context.WithCancel(ctx)
	go func() {
		time.Sleep(100 * time.Millisecond)
		stop()
	}()
	err = New("cat-file", WithFlag("--batch")).Run(stopCtx,
		WithDir(repoDir), WithStdin(blockingReader{done: stopCtx.Done()}))
	require.Error(t, err)

	require.NoError(t, New("remote", WithAction("add"), WithArg("origin"), WithArg("https://example.com/repo.git")).
		Run(ctx, WithDir(repoDir)))

	parent.End()

	var spans []sdktrace.ReadOnlySpan
	for _, span := range recorder.Ended() {
		if span.SpanContext().TraceID() == parent.SpanContext().TraceID() {
			spans = append(spans, span)
		}
	}
	require.Len(t, spans, 7)

	version, config, timedOut, canceled, stopped, remoteAdd := spans[0], spans[1], spans[2], spans[3], spans[4], spans[5]

	assert.Equal(t, "git version", version.Name())
	assert.Equal(t, parent.SpanContext().SpanID(), version.Parent().SpanID())
	assert.Contains(t, version.Attributes(), semconv.ProcessExitCode(0))
	assert.Equal(t, codes.Unset, version.Status().Code)

	assert.Equal(t, "git config", config.Name())
	assert.Contains(t, config.Attributes(), semconv.ProcessExitCode(1))
	assert.Equal(t, codes.Unset, config.Status().Code)

	assert.Equal(t, "git version", timedOut.Name())
	assert.Equal(t, codes.Error, timedOut.Status().Code)

	assert.Equal(t, "git version", canceled.Name())
	assert.Equal(t, codes.Unset, canceled.Status().Code)

	assert.Equal(t, "git cat-file", stopped.Name())
	assert.Equal(t, codes.Unset, stopped.Status().Code)

	assert.Equal(t, "git remote add", remoteAdd.Name())
	assert.Equal(t, codes.Unset, remoteAdd.Status().Code)
}

// blockingReader keeps stdin open until done is closed, like a cat-file --batch caller waiting for more input.
type blockingReader struct {
	done <-chan struct{}
}

func (r blockingReader) Read([]byte) (int, error) {
	<-r.done
	return 0, io.EOF
}
