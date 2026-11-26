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

package render

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errCt5Write = errors.New("write failed")

// ct5FailingWriter fails every write after the first okWrites calls.
type ct5FailingWriter struct {
	okWrites int
	writes   int
}

func (w *ct5FailingWriter) Write(b []byte) (int, error) {
	w.writes++
	if w.writes > w.okWrites {
		return 0, errCt5Write
	}
	return len(b), nil
}

type ct5Flusher struct{ flushed int }

func (f *ct5Flusher) Flush() { f.flushed++ }

func TestCt5SSEStreamEventWriteFailures(t *testing.T) {
	event := &sse.Event{Type: enum.SSETypeExecutionRunning, Data: json.RawMessage(`{"a":1}`)}

	// event: header, data: header, encoded payload, trailing newline.
	for okWrites := 0; okWrites < 4; okWrites++ {
		w := &ct5FailingWriter{okWrites: okWrites}
		stream := sseStream{enc: json.NewEncoder(w), writer: w, flusher: &ct5Flusher{}}

		err := stream.event(event)
		require.ErrorIs(t, err, errCt5Write, "okWrites=%d", okWrites)
	}

	flusher := &ct5Flusher{}
	w := &ct5FailingWriter{okWrites: 100}
	require.NoError(t, sseStream{enc: json.NewEncoder(w), writer: w, flusher: flusher}.event(event))
	require.Equal(t, 1, flusher.flushed)
}

func TestCt5SSEStreamCloseAndPingFailures(t *testing.T) {
	w := &ct5FailingWriter{}
	stream := sseStream{enc: json.NewEncoder(w), writer: w, flusher: &ct5Flusher{}}
	require.ErrorIs(t, stream.close(), errCt5Write)
	require.ErrorIs(t, stream.ping(), errCt5Write)

	rec := httptest.NewRecorder()
	ok := sseStream{enc: json.NewEncoder(rec), writer: io.Writer(rec), flusher: rec}
	require.NoError(t, ok.close())
	require.NoError(t, ok.ping())
	require.Contains(t, rec.Body.String(), "event: error")
	require.Contains(t, rec.Body.String(), ": ping")
	require.Equal(t, http.StatusOK, rec.Code)
}
