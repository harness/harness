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

package types

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestMultiReadCloser_Read(t *testing.T) {
	content := "test content"
	reader := strings.NewReader(content)
	closeCalled := false

	mrc := &MultiReadCloser{
		Reader: reader,
	}

	// Test reading
	buf := make([]byte, len(content))
	n, err := mrc.Read(buf)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	if n != len(content) {
		t.Errorf("Read() returned %d bytes, expected %d", n, len(content))
	}

	if string(buf) != content {
		t.Errorf("Read() returned %q, expected %q", string(buf), content)
	}

	// Verify close wasn't called yet
	if closeCalled {
		t.Errorf("CloseFunc was called before Close()")
	}
}

func TestMultiReadCloser_Close(t *testing.T) {
	reader := strings.NewReader("test")
	closeCalled := false

	mrc := &MultiReadCloser{
		Reader: reader,
		CloseFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	// Test closing
	err := mrc.Close()
	if err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	if !closeCalled {
		t.Errorf("CloseFunc was not called")
	}
}

func TestMultiReadCloser_CloseError(t *testing.T) {
	reader := strings.NewReader("test")
	expectedErr := errors.New("close error")

	mrc := &MultiReadCloser{
		Reader: reader,
		CloseFunc: func() error {
			return expectedErr
		},
	}

	// Test closing with error
	err := mrc.Close()
	if !errors.Is(err, expectedErr) {
		t.Errorf("Close() returned error %v, expected %v", err, expectedErr)
	}
}

func TestMultiReadCloser_ReadAndClose(t *testing.T) {
	content := "hello world"
	reader := strings.NewReader(content)
	closeCalled := false

	mrc := &MultiReadCloser{
		Reader: reader,
		CloseFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	// Read all content
	buf := make([]byte, len(content))
	n, err := io.ReadFull(mrc, buf)
	if err != nil {
		t.Fatalf("ReadFull() returned error: %v", err)
	}

	if n != len(content) {
		t.Errorf("ReadFull() returned %d bytes, expected %d", n, len(content))
	}

	if string(buf) != content {
		t.Errorf("ReadFull() returned %q, expected %q", string(buf), content)
	}

	// Close
	err = mrc.Close()
	if err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	if !closeCalled {
		t.Errorf("CloseFunc was not called")
	}
}

func TestMultiReadCloser_MultipleReads(t *testing.T) {
	content := "test content for multiple reads"
	reader := strings.NewReader(content)

	mrc := &MultiReadCloser{
		Reader: reader,
	}

	// Read in chunks
	buf1 := make([]byte, 4)
	n1, err := mrc.Read(buf1)
	if err != nil {
		t.Fatalf("First Read() returned error: %v", err)
	}
	if n1 != 4 {
		t.Errorf("First Read() returned %d bytes, expected 4", n1)
	}

	buf2 := make([]byte, 8)
	n2, err := mrc.Read(buf2)
	if err != nil {
		t.Fatalf("Second Read() returned error: %v", err)
	}
	if n2 != 8 {
		t.Errorf("Second Read() returned %d bytes, expected 8", n2)
	}

	// Verify content
	combined := string(buf1) + string(buf2)
	if combined != content[:12] {
		t.Errorf("Combined reads returned %q, expected %q", combined, content[:12])
	}
}

func TestMultiReadCloser_EOF(t *testing.T) {
	content := "short"
	reader := strings.NewReader(content)

	mrc := &MultiReadCloser{
		Reader: reader,
		CloseFunc: func() error {
			return nil
		},
	}

	// Read all content
	buf := make([]byte, len(content))
	_, err := io.ReadFull(mrc, buf)
	if err != nil {
		t.Fatalf("ReadFull() returned error: %v", err)
	}

	// Try to read more - should get EOF
	buf2 := make([]byte, 10)
	n, err := mrc.Read(buf2)
	if !errors.Is(err, io.EOF) {
		t.Errorf("Read() after EOF returned error %v, expected io.EOF", err)
	}
	if n != 0 {
		t.Errorf("Read() after EOF returned %d bytes, expected 0", n)
	}
}

func TestMultiReadCloser_NilCloseFunc(t *testing.T) {
	reader := strings.NewReader("test")

	mrc := &MultiReadCloser{
		Reader: reader,
	}

	// This should panic when Close() is called
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Close() with nil CloseFunc should panic")
		}
	}()

	mrc.Close()
}

func TestMultiReadCloser_EmptyReader(t *testing.T) {
	reader := strings.NewReader("")
	closeCalled := false

	mrc := &MultiReadCloser{
		Reader: reader,
		CloseFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	// Try to read from empty reader
	buf := make([]byte, 10)
	n, err := mrc.Read(buf)
	if !errors.Is(err, io.EOF) {
		t.Errorf("Read() from empty reader returned error %v, expected io.EOF", err)
	}
	if n != 0 {
		t.Errorf("Read() from empty reader returned %d bytes, expected 0", n)
	}

	// Close should still work
	err = mrc.Close()
	if err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	if !closeCalled {
		t.Errorf("CloseFunc was not called")
	}
}

func TestMultiReadCloser_MultipleCloses(t *testing.T) {
	reader := strings.NewReader("test")
	closeCount := 0

	mrc := &MultiReadCloser{
		Reader: reader,
		CloseFunc: func() error {
			closeCount++
			return nil
		},
	}

	// Close multiple times
	err := mrc.Close()
	if err != nil {
		t.Fatalf("First Close() returned error: %v", err)
	}

	err = mrc.Close()
	if err != nil {
		t.Fatalf("Second Close() returned error: %v", err)
	}

	// Verify CloseFunc was called twice
	if closeCount != 2 {
		t.Errorf("CloseFunc was called %d times, expected 2", closeCount)
	}
}
