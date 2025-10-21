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

package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestWithRequestID(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
	}{
		{
			name:      "normal request ID",
			requestID: "req-123456",
		},
		{
			name:      "empty request ID",
			requestID: "",
		},
		{
			name:      "UUID request ID",
			requestID: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:      "long request ID",
			requestID: strings.Repeat("a", 100),
		},
		{
			name:      "special characters",
			requestID: "req-123!@#$%^&*()",
		},
		{
			name:      "unicode request ID",
			requestID: "req-世界-🌍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture log output
			var buf bytes.Buffer
			logger := zerolog.New(&buf)

			// Apply the WithRequestID option
			option := WithRequestID(tt.requestID)
			logCtx := logger.With()
			logCtx = option(logCtx)

			// Log a message to test the option
			logger = logCtx.Logger()
			logger.Info().Msg("test message")

			// Parse the log output
			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			if err != nil {
				t.Fatalf("Failed to parse log output: %v", err)
			}

			// Check if request_id is present and correct
			if requestID, ok := logEntry["request_id"]; ok {
				if requestID != tt.requestID {
					t.Errorf("Expected request_id %q, got %q", tt.requestID, requestID)
				}
			} else {
				t.Error("request_id field not found in log output")
			}
		})
	}
}

func TestUpdateContext(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create a context with the logger
	ctx := logger.WithContext(context.Background())

	// Test updating context with request ID
	UpdateContext(ctx, WithRequestID("test-req-123"))

	// Log a message using the updated context
	zerolog.Ctx(ctx).Info().Msg("test message")

	// Parse the log output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	// Check if request_id is present
	if requestID, ok := logEntry["request_id"]; ok {
		if requestID != "test-req-123" {
			t.Errorf("Expected request_id %q, got %q", "test-req-123", requestID)
		}
	} else {
		t.Error("request_id field not found in log output")
	}
}

func TestUpdateContextMultipleOptions(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create a context with the logger
	ctx := logger.WithContext(context.Background())

	// Create multiple options
	option1 := WithRequestID("req-456")
	option2 := func(c zerolog.Context) zerolog.Context {
		return c.Str("user_id", "user-789")
	}
	option3 := func(c zerolog.Context) zerolog.Context {
		return c.Int("version", 1)
	}

	// Update context with multiple options
	UpdateContext(ctx, option1, option2, option3)

	// Log a message using the updated context
	zerolog.Ctx(ctx).Info().Msg("test message")

	// Parse the log output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	// Check all fields are present
	if requestID, ok := logEntry["request_id"]; !ok || requestID != "req-456" {
		t.Errorf("Expected request_id %q, got %q", "req-456", requestID)
	}
	if userID, ok := logEntry["user_id"]; !ok || userID != "user-789" {
		t.Errorf("Expected user_id %q, got %q", "user-789", userID)
	}
	if version, ok := logEntry["version"]; !ok || version != float64(1) {
		t.Errorf("Expected version %v, got %v", 1, version)
	}
}

func TestNewContext(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create a parent context with the logger
	parentCtx := logger.WithContext(context.Background())

	// Create a new context with request ID
	childCtx := NewContext(parentCtx, WithRequestID("child-req-123"))

	// Verify that the parent context is not modified
	zerolog.Ctx(parentCtx).Info().Msg("parent message")

	// Parse the parent log output
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) > 0 {
		var parentLogEntry map[string]interface{}
		err := json.Unmarshal([]byte(lines[0]), &parentLogEntry)
		if err != nil {
			t.Fatalf("Failed to parse parent log output: %v", err)
		}

		// Parent should not have request_id
		if _, ok := parentLogEntry["request_id"]; ok {
			t.Error("Parent context should not have request_id")
		}
	}

	// Clear buffer for child test
	buf.Reset()

	// Log a message using the child context
	zerolog.Ctx(childCtx).Info().Msg("child message")

	// Parse the child log output
	var childLogEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &childLogEntry)
	if err != nil {
		t.Fatalf("Failed to parse child log output: %v", err)
	}

	// Child should have request_id
	if requestID, ok := childLogEntry["request_id"]; !ok || requestID != "child-req-123" {
		t.Errorf("Expected child request_id %q, got %q", "child-req-123", requestID)
	}
}

func TestNewContextMultipleOptions(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create a parent context with the logger
	parentCtx := logger.WithContext(context.Background())

	// Create multiple options
	option1 := WithRequestID("new-req-789")
	option2 := func(c zerolog.Context) zerolog.Context {
		return c.Str("service", "test-service")
	}
	option3 := func(c zerolog.Context) zerolog.Context {
		return c.Bool("debug", true)
	}

	// Create a new context with multiple options
	childCtx := NewContext(parentCtx, option1, option2, option3)

	// Log a message using the child context
	zerolog.Ctx(childCtx).Info().Msg("test message")

	// Parse the log output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	// Check all fields are present
	if requestID, ok := logEntry["request_id"]; !ok || requestID != "new-req-789" {
		t.Errorf("Expected request_id %q, got %q", "new-req-789", requestID)
	}
	if service, ok := logEntry["service"]; !ok || service != "test-service" {
		t.Errorf("Expected service %q, got %q", "test-service", service)
	}
	if debug, ok := logEntry["debug"]; !ok || debug != true {
		t.Errorf("Expected debug %v, got %v", true, debug)
	}
}

func TestNewContextIsolation(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create a parent context with the logger
	parentCtx := logger.WithContext(context.Background())

	// Create two child contexts with different request IDs
	child1Ctx := NewContext(parentCtx, WithRequestID("child1-req"))
	child2Ctx := NewContext(parentCtx, WithRequestID("child2-req"))

	// Log from child1
	zerolog.Ctx(child1Ctx).Info().Msg("child1 message")

	// Parse child1 log
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var child1LogEntry map[string]interface{}
	err := json.Unmarshal([]byte(lines[0]), &child1LogEntry)
	if err != nil {
		t.Fatalf("Failed to parse child1 log output: %v", err)
	}

	if requestID, ok := child1LogEntry["request_id"]; !ok || requestID != "child1-req" {
		t.Errorf("Expected child1 request_id %q, got %q", "child1-req", requestID)
	}

	// Clear buffer
	buf.Reset()

	// Log from child2
	zerolog.Ctx(child2Ctx).Info().Msg("child2 message")

	// Parse child2 log
	var child2LogEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &child2LogEntry)
	if err != nil {
		t.Fatalf("Failed to parse child2 log output: %v", err)
	}

	if requestID, ok := child2LogEntry["request_id"]; !ok || requestID != "child2-req" {
		t.Errorf("Expected child2 request_id %q, got %q", "child2-req", requestID)
	}
}

func TestCustomOption(t *testing.T) {
	// Create a custom option
	customOption := func(c zerolog.Context) zerolog.Context {
		return c.Str("custom_field", "custom_value").Int("number", 42)
	}

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create a context with the logger
	ctx := logger.WithContext(context.Background())

	// Create a new context with custom option
	newCtx := NewContext(ctx, customOption)

	// Log a message using the new context
	zerolog.Ctx(newCtx).Info().Msg("test message")

	// Parse the log output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	// Check custom fields are present
	if customField, ok := logEntry["custom_field"]; !ok || customField != "custom_value" {
		t.Errorf("Expected custom_field %q, got %q", "custom_value", customField)
	}
	if number, ok := logEntry["number"]; !ok || number != float64(42) {
		t.Errorf("Expected number %v, got %v", 42, number)
	}
}

func TestEmptyOptions(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create a context with the logger
	ctx := logger.WithContext(context.Background())

	// Test UpdateContext with no options
	UpdateContext(ctx)

	// Test NewContext with no options
	newCtx := NewContext(ctx)

	// Log messages from both contexts
	zerolog.Ctx(ctx).Info().Msg("original context")
	buf.Reset()
	zerolog.Ctx(newCtx).Info().Msg("new context")

	// Both should work without errors
	if buf.Len() == 0 {
		t.Error("Expected log output from new context")
	}
}

// Benchmark tests.
func BenchmarkWithRequestID(b *testing.B) {
	logger := zerolog.New(bytes.NewBuffer(nil))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		option := WithRequestID("benchmark-req-123")
		logCtx := logger.With()
		_ = option(logCtx)
	}
}

func BenchmarkUpdateContext(b *testing.B) {
	logger := zerolog.New(bytes.NewBuffer(nil))
	ctx := logger.WithContext(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UpdateContext(ctx, WithRequestID("benchmark-req-123"))
	}
}

func BenchmarkNewContext(b *testing.B) {
	logger := zerolog.New(bytes.NewBuffer(nil))
	ctx := logger.WithContext(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewContext(ctx, WithRequestID("benchmark-req-123"))
	}
}
