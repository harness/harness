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

package usage

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestSv5WriteCounter(t *testing.T) {
	rec := httptest.NewRecorder()
	w := newWriter(rec)

	w.Header().Set("X-Test", "yes")
	w.WriteHeader(http.StatusCreated)
	n, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	_, err = w.Write([]byte(" world"))
	require.NoError(t, err)

	require.Equal(t, int64(11), w.n)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "yes", rec.Header().Get("X-Test"))
	require.Equal(t, "hello world", rec.Body.String())
}

type sv5Closer struct {
	io.Reader

	closed bool
}

func (c *sv5Closer) Close() error {
	c.closed = true
	return nil
}

func TestSv5ReadCounter(t *testing.T) {
	src := &sv5Closer{Reader: strings.NewReader("payload")}
	r := newReader(src)

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "payload", string(data))
	require.Equal(t, int64(7), r.n)

	require.NoError(t, r.Close())
	require.True(t, src.closed)
}

func TestSv5NewConfig(t *testing.T) {
	global := &types.Config{}
	global.UsageMetrics.FlushInterval = MinFlushInterval / 2
	global.UsageMetrics.QueueSize = 0

	cfg := NewConfig(global)
	require.Equal(t, MinFlushInterval, cfg.FlushInterval, "flush interval is raised to the minimum")
	require.Equal(t, QueueSize, cfg.QueueSize, "queue size falls back to the default")

	global.UsageMetrics.FlushInterval = time.Hour
	global.UsageMetrics.QueueSize = 17
	cfg = NewConfig(global)
	require.Equal(t, time.Hour, cfg.FlushInterval)
	require.Equal(t, 17, cfg.QueueSize)
}
