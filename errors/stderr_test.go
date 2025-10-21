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

package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "simple error message",
			text: "test error",
		},
		{
			name: "empty error message",
			text: "",
		},
		{
			name: "long error message",
			text: "this is a very long error message that contains multiple words and should be handled correctly",
		},
		{
			name: "error with special characters",
			text: "error with special chars: !@#$%^&*()",
		},
		{
			name: "error with unicode",
			text: "error with unicode: 世界 🌍",
		},
		{
			name: "error with newlines",
			text: "error\nwith\nnewlines",
		},
		{
			name: "error with tabs",
			text: "error\twith\ttabs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.text)
			if err == nil {
				t.Error("Expected error to be non-nil")
			}
			if err.Error() != tt.text {
				t.Errorf("Expected error message %q, got %q", tt.text, err.Error())
			}
		})
	}
}

func TestNewComparison(t *testing.T) {
	// Test that New creates errors that can be compared
	err1 := New("test error")
	err2 := New("test error")
	err3 := New("different error")

	// Different instances with same message should not be equal
	if errors.Is(err1, err2) {
		t.Error("Expected different error instances to not be equal")
	}

	// Different messages should not be equal
	if errors.Is(err1, err3) {
		t.Error("Expected errors with different messages to not be equal")
	}

	// But their messages should be the same
	if err1.Error() != err2.Error() {
		t.Error("Expected error messages to be the same")
	}
}

func TestIs(t *testing.T) {
	baseErr := New("base error")
	wrappedErr := errors.New("wrapped: " + baseErr.Error())
	differentErr := New("different error")

	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "same error",
			err:      baseErr,
			target:   baseErr,
			expected: true,
		},
		{
			name:     "different errors",
			err:      baseErr,
			target:   differentErr,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			target:   baseErr,
			expected: false,
		},
		{
			name:     "nil target",
			err:      baseErr,
			target:   nil,
			expected: false,
		},
		{
			name:     "both nil",
			err:      nil,
			target:   nil,
			expected: true,
		},
		{
			name:     "wrapped error",
			err:      wrappedErr,
			target:   baseErr,
			expected: false, // Our wrapper doesn't implement Unwrap
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Is(tt.err, tt.target)
			if result != tt.expected {
				t.Errorf("Expected Is(%v, %v) to be %v, got %v", tt.err, tt.target, tt.expected, result)
			}
		})
	}
}

// Custom error types for testing.
type customError struct {
	msg string
}

func (e customError) Error() string { return e.msg }

type anotherError struct {
	code int
}

func (e anotherError) Error() string { return "another error" }

func TestAs(t *testing.T) {
	customErr := customError{msg: "custom error"}
	anotherErr := anotherError{code: 123}
	standardErr := New("standard error")

	tests := []struct {
		name     string
		err      error
		target   interface{}
		expected bool
	}{
		{
			name:     "custom error to custom error",
			err:      customErr,
			target:   &customError{},
			expected: true,
		},
		{
			name:     "custom error to different type",
			err:      customErr,
			target:   &anotherError{},
			expected: false,
		},
		{
			name:     "standard error to custom type",
			err:      standardErr,
			target:   &customError{},
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			target:   &customError{},
			expected: false,
		},
		{
			name:     "another error type",
			err:      anotherErr,
			target:   &anotherError{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := As(tt.err, tt.target)
			if result != tt.expected {
				t.Errorf("Expected As(%v, %T) to be %v, got %v", tt.err, tt.target, tt.expected, result)
			}
		})
	}
}

// Custom error type for TestAsWithValues.
type codeError struct {
	msg  string
	code int
}

func (e codeError) Error() string { return e.msg }

func TestAsWithValues(t *testing.T) {
	// Test that As correctly populates the target

	originalErr := codeError{msg: "test error", code: 42}
	var target codeError

	result := As(originalErr, &target)
	if !result {
		t.Error("Expected As to return true")
	}

	if target.msg != originalErr.msg {
		t.Errorf("Expected target msg to be %q, got %q", originalErr.msg, target.msg)
	}

	if target.code != originalErr.code {
		t.Errorf("Expected target code to be %d, got %d", originalErr.code, target.code)
	}
}

// Benchmark tests.
func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New("benchmark error")
	}
}

func BenchmarkIs(b *testing.B) {
	err1 := New("error 1")
	err2 := New("error 2")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Is(err1, err2)
	}
}

func BenchmarkAs(b *testing.B) {
	err := customError{msg: "test"}
	var target customError

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		As(err, &target)
	}
}
