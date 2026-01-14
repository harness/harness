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

package usage

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_writeCounter_Write(t *testing.T) {
	size := 1 << 16

	// Create a buffer to hold the payload.
	buffer := httptest.NewRecorder()
	writer := newWriter(buffer)

	expected := &bytes.Buffer{}
	for i := 0; i < size; i += sampleLength {
		if size-i < sampleLength {
			// Write only the remaining characters to reach the exact size.
			_, _ = writer.Write([]byte(sampleText[:size-i]))
			expected.WriteString(sampleText[:size-i])
			break
		}
		_, _ = writer.Write([]byte(sampleText))
		expected.WriteString(sampleText)
	}

	require.Equal(t, int64(size), writer.n, "expected %d, got %d", size, writer.n)
	require.Equal(t, expected.Bytes(), buffer.Body.Bytes())
}

func Test_readCounter_Read(t *testing.T) {
	size := 1 << 16

	buffer := &bytes.Buffer{}
	reader := newReader(io.NopCloser(buffer))

	for i := 0; i < size; i += sampleLength {
		if size-i < sampleLength {
			// Write only the remaining characters to reach the exact size.
			buffer.WriteString(sampleText[:size-i])
			break
		}
		buffer.WriteString(sampleText)
	}

	expected := buffer.Bytes()
	got := &bytes.Buffer{}

	_, err := io.Copy(got, reader)
	require.NoError(t, err)

	require.Equal(t, int64(size), reader.n, "expected %d, got %d", size, reader.n)
	require.Equal(t, expected, got.Bytes())
}
