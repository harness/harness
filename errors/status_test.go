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
)

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{"StatusConflict", StatusConflict, "conflict"},
		{"StatusInternal", StatusInternal, "internal"},
		{"StatusInvalidArgument", StatusInvalidArgument, "invalid"},
		{"StatusNotFound", StatusNotFound, "not_found"},
		{"StatusNotImplemented", StatusNotImplemented, "not_implemented"},
		{"StatusUnauthorized", StatusUnauthorized, "unauthorized"},
		{"StatusForbidden", StatusForbidden, "forbidden"},
		{"StatusFailed", StatusFailed, "failed"},
		{"StatusPreconditionFailed", StatusPreconditionFailed, "precondition_failed"},
		{"StatusAborted", StatusAborted, "aborted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, string(tt.status))
			}
		})
	}
}

func TestErrorStruct(t *testing.T) {
	err := &Error{
		Status:  StatusNotFound,
		Message: "resource not found",
		Err:     errors.New("underlying error"),
		Details: map[string]any{"resource_id": "123"},
	}

	// Test Error() method
	expectedMsg := "resource not found: underlying error"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}

	// Test Unwrap() method
	if err.Unwrap() == nil {
		t.Error("Expected Unwrap() to return non-nil error")
	}
	if err.Unwrap().Error() != "underlying error" {
		t.Errorf("Expected unwrapped error to be %q, got %q", "underlying error", err.Unwrap().Error())
	}
}

func TestErrorWithoutUnderlyingError(t *testing.T) {
	err := &Error{
		Status:  StatusInvalidArgument,
		Message: "invalid input",
	}

	// Test Error() method without underlying error
	if err.Error() != "invalid input" {
		t.Errorf("Expected error message %q, got %q", "invalid input", err.Error())
	}

	// Test Unwrap() method
	if err.Unwrap() != nil {
		t.Error("Expected Unwrap() to return nil when no underlying error")
	}
}

func TestErrorSetErr(t *testing.T) {
	err := &Error{
		Status:  StatusInternal,
		Message: "internal error",
	}

	underlyingErr := errors.New("database connection failed")
	result := err.SetErr(underlyingErr)

	// Should return the same error instance
	if result != err {
		t.Error("Expected SetErr to return the same error instance")
	}

	// Should set the underlying error
	if !errors.Is(err.Err, underlyingErr) {
		t.Error("Expected SetErr to set the underlying error")
	}
}

func TestErrorSetDetails(t *testing.T) {
	err := &Error{
		Status:  StatusNotFound,
		Message: "user not found",
	}

	details := map[string]any{
		"user_id": "123",
		"table":   "users",
	}
	result := err.SetDetails(details)

	// Should return the same error instance
	if result != err {
		t.Error("Expected SetDetails to return the same error instance")
	}

	// Should set the details
	if err.Details == nil {
		t.Error("Expected SetDetails to set the details")
	}
	if err.Details["user_id"] != "123" {
		t.Error("Expected details to contain user_id")
	}
	if err.Details["table"] != "users" {
		t.Error("Expected details to contain table")
	}
}

func TestAsStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected Status
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "Error with status",
			err:      &Error{Status: StatusNotFound, Message: "not found"},
			expected: StatusNotFound,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: StatusInternal,
		},
		{
			name:     "wrapped Error",
			err:      fmt.Errorf("wrapped: %w", &Error{Status: StatusConflict, Message: "conflict"}),
			expected: StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AsStatus(tt.err)
			if result != tt.expected {
				t.Errorf("Expected AsStatus(%v) to be %q, got %q", tt.err, tt.expected, result)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "Error with message",
			err:      &Error{Status: StatusNotFound, Message: "resource not found"},
			expected: "resource not found",
		},
		{
			name:     "standard error",
			err:      errors.New("standard error message"),
			expected: "standard error message",
		},
		{
			name:     "wrapped Error",
			err:      fmt.Errorf("wrapped: %w", &Error{Status: StatusConflict, Message: "conflict occurred"}),
			expected: "conflict occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Message(tt.err)
			if result != tt.expected {
				t.Errorf("Expected Message(%v) to be %q, got %q", tt.err, tt.expected, result)
			}
		})
	}
}

func TestDetails(t *testing.T) {
	details := map[string]any{"key": "value", "number": 42}

	tests := []struct {
		name     string
		err      error
		expected map[string]any
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: nil,
		},
		{
			name:     "Error with details",
			err:      &Error{Status: StatusNotFound, Message: "not found", Details: details},
			expected: details,
		},
		{
			name:     "Error without details",
			err:      &Error{Status: StatusNotFound, Message: "not found"},
			expected: nil,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: nil,
		},
		{
			name:     "wrapped Error with details",
			err:      fmt.Errorf("wrapped: %w", &Error{Status: StatusConflict, Message: "conflict", Details: details}),
			expected: details,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Details(tt.err)
			if !mapsEqual(result, tt.expected) {
				t.Errorf("Expected Details(%v) to be %v, got %v", tt.err, tt.expected, result)
			}
		})
	}
}

func TestAsError(t *testing.T) {
	appErr := &Error{Status: StatusNotFound, Message: "not found"}
	stdErr := errors.New("standard error")

	tests := []struct {
		name     string
		err      error
		expected *Error
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: nil,
		},
		{
			name:     "Error type",
			err:      appErr,
			expected: appErr,
		},
		{
			name:     "standard error",
			err:      stdErr,
			expected: nil,
		},
		{
			name:     "wrapped Error",
			err:      fmt.Errorf("wrapped: %w", appErr),
			expected: appErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AsError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected AsError(%v) to be %v, got %v", tt.err, tt.expected, result)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		format   string
		args     []any
		expected *Error
	}{
		{
			name:     "simple format",
			status:   StatusNotFound,
			format:   "user not found",
			args:     nil,
			expected: &Error{Status: StatusNotFound, Message: "user not found"},
		},
		{
			name:     "format with args",
			status:   StatusInvalidArgument,
			format:   "invalid user ID: %d",
			args:     []any{123},
			expected: &Error{Status: StatusInvalidArgument, Message: "invalid user ID: 123"},
		},
		{
			name:     "format with multiple args",
			status:   StatusConflict,
			format:   "user %s already exists with email %s",
			args:     []any{"john", "john@example.com"},
			expected: &Error{Status: StatusConflict, Message: "user john already exists with email john@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Format(tt.status, tt.format, tt.args...)
			if result.Status != tt.expected.Status {
				t.Errorf("Expected status %q, got %q", tt.expected.Status, result.Status)
			}
			if result.Message != tt.expected.Message {
				t.Errorf("Expected message %q, got %q", tt.expected.Message, result.Message)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string, ...any) *Error
		status   Status
		format   string
		args     []any
		expected string
	}{
		{"NotFound", NotFoundf, StatusNotFound, "user %d not found", []any{123}, "user 123 not found"},
		{"InvalidArgument", InvalidArgumentf, StatusInvalidArgument,
			"invalid email: %s", []any{"invalid"}, "invalid email: invalid"},
		{"Conflict", Conflictf, StatusConflict, "user %s exists", []any{"john"}, "user john exists"},
		{"PreconditionFailed", PreconditionFailedf, StatusPreconditionFailed, "version mismatch", nil, "version mismatch"},
		{"Unauthorized", Unauthorizedf, StatusUnauthorized, "invalid token", nil, "invalid token"},
		{"Forbidden", Forbiddenf, StatusForbidden, "access denied", nil, "access denied"},
		{"Failed", Failedf, StatusFailed, "operation failed", nil, "operation failed"},
		{"Aborted", Abortedf, StatusAborted, "operation aborted", nil, "operation aborted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.format, tt.args...)
			if result.Status != tt.status {
				t.Errorf("Expected status %q, got %q", tt.status, result.Status)
			}
			if result.Message != tt.expected {
				t.Errorf("Expected message %q, got %q", tt.expected, result.Message)
			}
		})
	}
}

func TestInternal(t *testing.T) {
	underlyingErr := errors.New("database connection failed")
	result := Internalf(underlyingErr, "failed to get user %d", 123)

	if result.Status != StatusInternal {
		t.Errorf("Expected status %q, got %q", StatusInternal, result.Status)
	}

	expectedMsg := "failed to get user 123"
	if result.Message != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, result.Message)
	}

	if result.Err == nil {
		t.Error("Expected underlying error to be set")
	}

	// The underlying error should be wrapped
	expectedErrMsg := "failed to get user 123: database connection failed"
	if result.Err.Error() != expectedErrMsg {
		t.Errorf("Expected underlying error message %q, got %q", expectedErrMsg, result.Err.Error())
	}
}

func TestStatusCheckFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(error) bool
		status   Status
		expected bool
	}{
		{"IsNotFound with NotFound", IsNotFound, StatusNotFound, true},
		{"IsNotFound with Conflict", IsNotFound, StatusConflict, false},
		{"IsConflict with Conflict", IsConflict, StatusConflict, true},
		{"IsConflict with NotFound", IsConflict, StatusNotFound, false},
		{"IsInvalidArgument with InvalidArgument", IsInvalidArgument, StatusInvalidArgument, true},
		{"IsInvalidArgument with Internal", IsInvalidArgument, StatusInternal, false},
		{"IsInternal with Internal", IsInternal, StatusInternal, true},
		{"IsInternal with NotFound", IsInternal, StatusNotFound, false},
		{"IsPreconditionFailed with PreconditionFailed", IsPreconditionFailed, StatusPreconditionFailed, true},
		{"IsPreconditionFailed with Aborted", IsPreconditionFailed, StatusAborted, false},
		{"IsAborted with Aborted", IsAborted, StatusAborted, true},
		{"IsAborted with Failed", IsAborted, StatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &Error{Status: tt.status, Message: "test error"}
			result := tt.fn(err)
			if result != tt.expected {
				t.Errorf("Expected %s(%v) to be %v, got %v", tt.name, err, tt.expected, result)
			}
		})
	}
}

func TestStatusCheckFunctionsWithStandardError(t *testing.T) {
	stdErr := errors.New("standard error")

	// All status check functions should return false for standard errors,
	// except IsInternal which should return true (since standard errors are treated as internal)
	tests := []struct {
		name     string
		fn       func(error) bool
		expected bool
	}{
		{"IsNotFound", IsNotFound, false},
		{"IsConflict", IsConflict, false},
		{"IsInvalidArgument", IsInvalidArgument, false},
		{"IsInternal", IsInternal, true}, // Standard errors are treated as internal
		{"IsPreconditionFailed", IsPreconditionFailed, false},
		{"IsAborted", IsAborted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(stdErr)
			if result != tt.expected {
				t.Errorf("Expected %s(standard error) to be %v, got %v", tt.name, tt.expected, result)
			}
		})
	}
}

func TestStatusCheckFunctionsWithNil(t *testing.T) {
	// All status check functions should return false for nil errors
	tests := []struct {
		name string
		fn   func(error) bool
	}{
		{"IsNotFound", IsNotFound},
		{"IsConflict", IsConflict},
		{"IsInvalidArgument", IsInvalidArgument},
		{"IsInternal", IsInternal},
		{"IsPreconditionFailed", IsPreconditionFailed},
		{"IsAborted", IsAborted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(nil)
			if result {
				t.Errorf("Expected %s(nil) to be false, got true", tt.name)
			}
		})
	}
}

// Helper function to compare maps.
func mapsEqual(a, b map[string]any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// Benchmark tests.
func BenchmarkErrorError(b *testing.B) {
	err := &Error{
		Status:  StatusNotFound,
		Message: "resource not found",
		Err:     errors.New("underlying error"),
	}

	for b.Loop() {
		_ = err.Error()
	}
}

func BenchmarkAsStatus(b *testing.B) {
	err := &Error{Status: StatusNotFound, Message: "not found"}

	for b.Loop() {
		AsStatus(err)
	}
}

func BenchmarkFormat(b *testing.B) {
	for b.Loop() {
		_ = Format(StatusNotFound, "user %d not found", 123)
	}
}

func BenchmarkIsNotFound(b *testing.B) {
	err := &Error{Status: StatusNotFound, Message: "not found"}

	for b.Loop() {
		IsNotFound(err)
	}
}
