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

package livelog

import (
	"reflect"
	"testing"
)

func TestLine(t *testing.T) {
	tests := []struct {
		name     string
		line     Line
		expected Line
	}{
		{
			name: "basic line",
			line: Line{
				Number:    1,
				Message:   "hello world",
				Timestamp: 1234567890,
			},
			expected: Line{
				Number:    1,
				Message:   "hello world",
				Timestamp: 1234567890,
			},
		},
		{
			name: "empty message",
			line: Line{
				Number:    0,
				Message:   "",
				Timestamp: 0,
			},
			expected: Line{
				Number:    0,
				Message:   "",
				Timestamp: 0,
			},
		},
		{
			name: "negative number",
			line: Line{
				Number:    -1,
				Message:   "error message",
				Timestamp: 9876543210,
			},
			expected: Line{
				Number:    -1,
				Message:   "error message",
				Timestamp: 9876543210,
			},
		},
		{
			name: "large timestamp",
			line: Line{
				Number:    999999,
				Message:   "large timestamp",
				Timestamp: 9223372036854775807, // max int64
			},
			expected: Line{
				Number:    999999,
				Message:   "large timestamp",
				Timestamp: 9223372036854775807,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			line := test.line

			if line.Number != test.expected.Number {
				t.Errorf("expected number %d, got %d", test.expected.Number, line.Number)
			}

			if line.Message != test.expected.Message {
				t.Errorf("expected message %q, got %q", test.expected.Message, line.Message)
			}

			if line.Timestamp != test.expected.Timestamp {
				t.Errorf("expected timestamp %d, got %d", test.expected.Timestamp, line.Timestamp)
			}
		})
	}
}

func TestLogStreamInfo(t *testing.T) {
	tests := []struct {
		name     string
		info     LogStreamInfo
		expected LogStreamInfo
	}{
		{
			name: "empty streams",
			info: LogStreamInfo{
				Streams: map[int64]int{},
			},
			expected: LogStreamInfo{
				Streams: map[int64]int{},
			},
		},
		{
			name: "single stream",
			info: LogStreamInfo{
				Streams: map[int64]int{
					1: 5,
				},
			},
			expected: LogStreamInfo{
				Streams: map[int64]int{
					1: 5,
				},
			},
		},
		{
			name: "multiple streams",
			info: LogStreamInfo{
				Streams: map[int64]int{
					1:  3,
					2:  7,
					10: 1,
				},
			},
			expected: LogStreamInfo{
				Streams: map[int64]int{
					1:  3,
					2:  7,
					10: 1,
				},
			},
		},
		{
			name: "streams with zero subscribers",
			info: LogStreamInfo{
				Streams: map[int64]int{
					1: 0,
					2: 5,
					3: 0,
				},
			},
			expected: LogStreamInfo{
				Streams: map[int64]int{
					1: 0,
					2: 5,
					3: 0,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := test.info

			if !reflect.DeepEqual(info.Streams, test.expected.Streams) {
				t.Errorf("expected streams %v, got %v", test.expected.Streams, info.Streams)
			}
		})
	}
}

func TestLogStreamInfo_NilStreams(t *testing.T) {
	info := LogStreamInfo{
		Streams: nil,
	}

	// Should be able to access nil map without panic
	if info.Streams != nil {
		t.Error("expected nil streams")
	}
}

func TestLine_JSONTags(t *testing.T) {
	// Test that the struct has the expected JSON tags
	lineType := reflect.TypeOf(Line{})

	// Check Number field
	numberField, found := lineType.FieldByName("Number")
	if !found {
		t.Fatal("Number field not found")
	}
	if tag := numberField.Tag.Get("json"); tag != "pos" {
		t.Errorf("expected Number field to have json tag 'pos', got %q", tag)
	}

	// Check Message field
	messageField, found := lineType.FieldByName("Message")
	if !found {
		t.Fatal("Message field not found")
	}
	if tag := messageField.Tag.Get("json"); tag != "out" {
		t.Errorf("expected Message field to have json tag 'out', got %q", tag)
	}

	// Check Timestamp field
	timestampField, found := lineType.FieldByName("Timestamp")
	if !found {
		t.Fatal("Timestamp field not found")
	}
	if tag := timestampField.Tag.Get("json"); tag != "time" {
		t.Errorf("expected Timestamp field to have json tag 'time', got %q", tag)
	}
}

func TestLogStreamInfo_JSONTags(t *testing.T) {
	// Test that the struct has the expected JSON tags
	infoType := reflect.TypeOf(LogStreamInfo{})

	// Check Streams field
	streamsField, found := infoType.FieldByName("Streams")
	if !found {
		t.Fatal("Streams field not found")
	}
	if tag := streamsField.Tag.Get("json"); tag != "streams" {
		t.Errorf("expected Streams field to have json tag 'streams', got %q", tag)
	}
}
