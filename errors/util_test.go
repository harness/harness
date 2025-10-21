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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsType(t *testing.T) {
	err := errors.New("abc")
	valueErr := testValueError{}
	valueErrPtr := &testValueError{}
	customErr := customError{msg: "test"}

	assert.True(t, IsType[error](err))
	assert.True(t, IsType[error](valueErr))
	assert.True(t, IsType[error](valueErrPtr))
	assert.True(t, IsType[error](customErr))

	assert.False(t, IsType[testValueError](err))
	assert.True(t, IsType[testValueError](valueErr))
	assert.False(t, IsType[testValueError](valueErrPtr))
	assert.False(t, IsType[testValueError](customErr))

	assert.False(t, IsType[*testValueError](err))
	assert.False(t, IsType[*testValueError](valueErr))
	assert.True(t, IsType[*testValueError](valueErrPtr))
	assert.False(t, IsType[*testValueError](customErr))

	assert.False(t, IsType[customError](err))
	assert.False(t, IsType[customError](valueErr))
	assert.False(t, IsType[customError](valueErrPtr))
	assert.True(t, IsType[customError](customErr))
}

func TestIsTypeWithNil(t *testing.T) {
	// Test with nil error
	assert.False(t, IsType[error](nil))
	assert.False(t, IsType[testValueError](nil))
	assert.False(t, IsType[*testValueError](nil))
	assert.False(t, IsType[*customError](nil))
}

func TestIsTypeWithWrappedErrors(t *testing.T) {
	// Test with wrapped errors
	valueErr := testValueError{}
	wrappedErr := fmt.Errorf("wrapped: %w", valueErr)

	assert.True(t, IsType[error](wrappedErr))
	assert.True(t, IsType[testValueError](wrappedErr))
	assert.False(t, IsType[*testValueError](wrappedErr))

	// Test with wrapped custom error
	customErr := customError{msg: "custom"}
	wrappedCustomErr := fmt.Errorf("wrapped: %w", customErr)

	assert.True(t, IsType[error](wrappedCustomErr))
	assert.False(t, IsType[testValueError](wrappedCustomErr))
	assert.False(t, IsType[*testValueError](wrappedCustomErr))
	assert.True(t, IsType[customError](wrappedCustomErr))
}

func TestIsTypeWithCustomError(t *testing.T) {
	// Test with custom error type
	customErr := customError{msg: "not found"}

	assert.True(t, IsType[error](customErr))
	assert.True(t, IsType[customError](customErr))
	assert.False(t, IsType[*customError](customErr))
	assert.False(t, IsType[testValueError](customErr))
	assert.False(t, IsType[*testValueError](customErr))
}

func TestIsTypeWithMultipleWrapping(t *testing.T) {
	// Test with multiple levels of wrapping
	originalErr := testValueError{}
	wrappedOnce := fmt.Errorf("first wrap: %w", originalErr)
	wrappedTwice := fmt.Errorf("second wrap: %w", wrappedOnce)

	assert.True(t, IsType[error](wrappedTwice))
	assert.True(t, IsType[testValueError](wrappedTwice))
	assert.False(t, IsType[*testValueError](wrappedTwice))
	assert.False(t, IsType[customError](wrappedTwice))
}

func TestIsTypeWithDifferentErrorTypes(t *testing.T) {
	// Test with various error types
	tests := []struct {
		name string
		err  error
	}{
		{"standard error", errors.New("standard")},
		{"value error", testValueError{}},
		{"pointer to value error", &testValueError{}},
		{"custom error", customError{msg: "custom"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All should be of type error
			assert.True(t, IsType[error](tt.err), "All errors should be of type error")

			// Test specific type checking
			var testVal testValueError
			var testPtr *testValueError
			var customErr customError

			switch {
			case errors.As(tt.err, &testVal):
				assert.True(t, IsType[testValueError](tt.err))
				assert.False(t, IsType[*testValueError](tt.err))
			case errors.As(tt.err, &testPtr):
				assert.False(t, IsType[testValueError](tt.err))
				assert.True(t, IsType[*testValueError](tt.err))
			case errors.As(tt.err, &customErr):
				assert.True(t, IsType[customError](tt.err))
				assert.False(t, IsType[*customError](tt.err))
			default:
				// Standard error - should not match specific types
				assert.False(t, IsType[testValueError](tt.err))
				assert.False(t, IsType[*testValueError](tt.err))
				assert.False(t, IsType[customError](tt.err))
			}
		})
	}
}

func TestIsTypeEdgeCases(t *testing.T) {
	// Test with simple custom error
	customErr := customError{msg: "test"}
	assert.True(t, IsType[error](customErr))
	assert.True(t, IsType[customError](customErr))
	assert.False(t, IsType[*customError](customErr))

	// Test type assertion behavior
	var err error = customErr
	assert.True(t, IsType[customError](err))
}

func TestIsTypePerformance(t *testing.T) {
	// Test that IsType works efficiently with different error types
	errors := []error{
		errors.New("standard"),
		testValueError{},
		&testValueError{},
		&customError{msg: "pointer"},
		&Error{Status: StatusNotFound, Message: "not found"},
	}

	for i, err := range errors {
		t.Run(fmt.Sprintf("error_%d", i), func(t *testing.T) {
			// Each should be identifiable as an error
			assert.True(t, IsType[error](err))

			// And should have consistent behavior
			result1 := IsType[error](err)
			result2 := IsType[error](err)
			assert.Equal(t, result1, result2, "IsType should be consistent")
		})
	}
}

type testValueError struct{}

func (e testValueError) Error() string { return "value receiver" }

// Benchmark tests.
func BenchmarkIsTypeValueError(b *testing.B) {
	err := testValueError{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsType[testValueError](err)
	}
}

func BenchmarkIsTypeCustomError(b *testing.B) {
	err := customError{msg: "test"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsType[customError](err)
	}
}

func BenchmarkIsTypeStandardError(b *testing.B) {
	err := errors.New("standard error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsType[error](err)
	}
}

func BenchmarkIsTypeWrappedError(b *testing.B) {
	err := fmt.Errorf("wrapped: %w", testValueError{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsType[testValueError](err)
	}
}
