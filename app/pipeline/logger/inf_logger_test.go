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

package logger

import (
	"bytes"
	"context"
	"errors"
	"testing"

	droneLogger "github.com/drone/runner-go/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func infContext(buf *bytes.Buffer) context.Context {
	l := zerolog.New(buf).Level(zerolog.TraceLevel)
	return l.WithContext(context.Background())
}

func TestInfWrappedZerolog_LogLevels(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := WithWrappedZerolog(infContext(buf))
	l := droneLogger.FromContext(ctx)

	l.Debug("d", 1)
	l.Debugf("d-%d", 2)
	l.Debugln("d3")
	l.Error("e1")
	l.Errorf("e-%d", 2)
	l.Errorln("e3")
	l.Info("i1")
	l.Infof("i-%d", 2)
	l.Infoln("i3")
	l.Trace("t1")
	l.Tracef("t-%d", 2)
	l.Traceln("t3")
	l.Warn("w1")
	l.Warnf("w-%d", 2)
	l.Warnln("w3")

	out := buf.String()
	for _, level := range []string{"debug", "error", "info", "trace", "warn"} {
		require.Contains(t, out, `"level":"`+level+`"`)
	}
	messages := []string{`"message":"d1"`, `"message":"d-2"`, `"message":"e-2"`, `"message":"w-2"`, `"message":"t-2"`}
	for _, msg := range messages {
		require.Contains(t, out, msg)
	}
}

func TestInfWrappedZerolog_WithErrorAndField(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := WithWrappedZerolog(infContext(buf))
	l := droneLogger.FromContext(ctx).WithError(errors.New("kaput")).WithField("stage", 3)
	l.Info("hello")

	out := buf.String()
	require.Contains(t, out, `"error":"kaput"`)
	require.Contains(t, out, `"stage":"3"`)
	require.Contains(t, out, `"message":"hello"`)
}

func TestInfWithUnwrappedZerolog(t *testing.T) {
	buf := &bytes.Buffer{}
	wrapped := WithWrappedZerolog(infContext(buf))
	unwrapped := WithUnwrappedZerolog(wrapped)
	zerolog.Ctx(unwrapped).Info().Msg("native")
	require.Contains(t, buf.String(), `"message":"native"`)

	// without a wrapped logger the global logger is used.
	ctx := WithUnwrappedZerolog(context.Background())
	require.NotNil(t, zerolog.Ctx(ctx))
}
