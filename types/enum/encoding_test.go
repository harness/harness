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

package enum

import (
	"reflect"
	"testing"
)

func TestContentEncodingTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		encoding ContentEncodingType
		expected string
	}{
		{
			name:     "UTF8 encoding",
			encoding: ContentEncodingTypeUTF8,
			expected: "utf8",
		},
		{
			name:     "Base64 encoding",
			encoding: ContentEncodingTypeBase64,
			expected: "base64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.encoding) != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, string(tt.encoding))
			}
		})
	}
}

func TestContentEncodingTypeString(t *testing.T) {
	tests := []struct {
		name     string
		encoding ContentEncodingType
		expected string
	}{
		{
			name:     "UTF8 string representation",
			encoding: ContentEncodingTypeUTF8,
			expected: "utf8",
		},
		{
			name:     "Base64 string representation",
			encoding: ContentEncodingTypeBase64,
			expected: "base64",
		},
		{
			name:     "Custom encoding",
			encoding: ContentEncodingType("custom"),
			expected: "custom",
		},
		{
			name:     "Empty encoding",
			encoding: ContentEncodingType(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(tt.encoding)
			if result != tt.expected {
				t.Errorf("Expected string representation to be %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestContentEncodingTypeEnum(t *testing.T) {
	// Test that Enum() returns the expected values
	encoding := ContentEncodingTypeUTF8
	enumValues := encoding.Enum()

	// Should return 2 values
	if len(enumValues) != 2 {
		t.Errorf("Expected Enum() to return 2 values, got %d", len(enumValues))
	}

	// Check that the values are correct
	expectedValues := []interface{}{
		ContentEncodingTypeBase64, // sorted order: base64 comes before utf8
		ContentEncodingTypeUTF8,
	}

	if !reflect.DeepEqual(enumValues, expectedValues) {
		t.Errorf("Expected Enum() to return %v, got %v", expectedValues, enumValues)
	}
}

func TestContentEncodingTypeEnumSorted(t *testing.T) {
	// Test that the enum values are sorted
	encoding := ContentEncodingTypeUTF8
	enumValues := encoding.Enum()

	// Convert back to ContentEncodingType for comparison
	var encodingTypes []ContentEncodingType
	for _, v := range enumValues {
		if enc, ok := v.(ContentEncodingType); ok {
			encodingTypes = append(encodingTypes, enc)
		}
	}

	// Check that they are in sorted order
	if len(encodingTypes) >= 2 {
		if encodingTypes[0] > encodingTypes[1] {
			t.Errorf("Expected enum values to be sorted, but %q > %q", encodingTypes[0], encodingTypes[1])
		}
	}
}

func TestContentEncodingTypeComparison(t *testing.T) {
	// Test string comparison
	if ContentEncodingTypeUTF8 == ContentEncodingTypeBase64 {
		t.Error("Expected UTF8 and Base64 encodings to be different")
	}

	if ContentEncodingTypeUTF8 < ContentEncodingTypeBase64 {
		t.Error("Expected UTF8 to be greater than Base64 in string comparison")
	}

	// Test equality
	utf8Copy := ContentEncodingType("utf8")
	if ContentEncodingTypeUTF8 != utf8Copy {
		t.Error("Expected identical encoding types to be equal")
	}
}

func TestContentEncodingTypeZeroValue(t *testing.T) {
	var encoding ContentEncodingType
	if encoding != "" {
		t.Errorf("Expected zero value of ContentEncodingType to be empty string, got %q", encoding)
	}
}

func TestContentEncodingTypeConversion(t *testing.T) {
	// Test conversion from string
	str := "utf8"
	encoding := ContentEncodingType(str)
	if encoding != ContentEncodingTypeUTF8 {
		t.Errorf("Expected conversion from string %q to give %q, got %q", str, ContentEncodingTypeUTF8, encoding)
	}

	// Test conversion to string
	result := string(ContentEncodingTypeBase64)
	if result != "base64" {
		t.Errorf("Expected conversion to string to give %q, got %q", "base64", result)
	}
}

func TestContentEncodingTypeValidation(t *testing.T) {
	// Test validation against known values
	validEncodings := []ContentEncodingType{
		ContentEncodingTypeUTF8,
		ContentEncodingTypeBase64,
	}

	for _, encoding := range validEncodings {
		t.Run(string(encoding), func(t *testing.T) {
			// Check that the encoding is in the enum
			enumValues := encoding.Enum()
			found := false
			for _, v := range enumValues {
				if v == encoding {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected %q to be found in enum values", encoding)
			}
		})
	}
}

func TestContentEncodingTypeInvalidValues(t *testing.T) {
	// Test with invalid/unknown encoding types
	invalidEncodings := []ContentEncodingType{
		ContentEncodingType("invalid"),
		ContentEncodingType("unknown"),
		ContentEncodingType("UTF8"),    // case sensitive
		ContentEncodingType("BASE64"),  // case sensitive
		ContentEncodingType("utf-8"),   // different format
		ContentEncodingType("base-64"), // different format
	}

	for _, encoding := range invalidEncodings {
		t.Run(string(encoding), func(t *testing.T) {
			// These should not be in the enum
			enumValues := encoding.Enum()
			found := false
			for _, v := range enumValues {
				if v == encoding {
					found = true
					break
				}
			}
			if found {
				t.Errorf("Expected %q to NOT be found in enum values", encoding)
			}
		})
	}
}

// Benchmark tests.
func BenchmarkContentEncodingTypeString(b *testing.B) {
	encoding := ContentEncodingTypeUTF8
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = string(encoding)
	}
}

func BenchmarkContentEncodingTypeEnum(b *testing.B) {
	encoding := ContentEncodingTypeUTF8
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = encoding.Enum()
	}
}

func BenchmarkContentEncodingTypeComparison(b *testing.B) {
	encoding1 := ContentEncodingTypeUTF8
	encoding2 := ContentEncodingTypeBase64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = encoding1 == encoding2
	}
}
