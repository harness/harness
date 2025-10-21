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
	"testing"
)

func TestSanitizeString(t *testing.T) {
	// Test with string type
	allValues := func() ([]string, string) {
		return []string{"apple", "banana", "cherry"}, "apple"
	}

	tests := []struct {
		name           string
		element        string
		expectedResult string
		expectedFound  bool
	}{
		{
			name:           "valid element",
			element:        "banana",
			expectedResult: "banana",
			expectedFound:  true,
		},
		{
			name:           "empty element returns default",
			element:        "",
			expectedResult: "apple",
			expectedFound:  true,
		},
		{
			name:           "invalid element returns default",
			element:        "grape",
			expectedResult: "apple",
			expectedFound:  false,
		},
		{
			name:           "first element",
			element:        "apple",
			expectedResult: "apple",
			expectedFound:  true,
		},
		{
			name:           "last element",
			element:        "cherry",
			expectedResult: "cherry",
			expectedFound:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := Sanitize(tt.element, allValues)
			if result != tt.expectedResult {
				t.Errorf("Expected result %q, got %q", tt.expectedResult, result)
			}
			if found != tt.expectedFound {
				t.Errorf("Expected found %v, got %v", tt.expectedFound, found)
			}
		})
	}
}

func TestSanitizeInt(t *testing.T) {
	// Test with int type
	allValues := func() ([]int, int) {
		return []int{1, 3, 5, 7, 9}, 1
	}

	tests := []struct {
		name           string
		element        int
		expectedResult int
		expectedFound  bool
	}{
		{
			name:           "valid element",
			element:        5,
			expectedResult: 5,
			expectedFound:  true,
		},
		{
			name:           "zero element returns default",
			element:        0,
			expectedResult: 1,
			expectedFound:  true,
		},
		{
			name:           "invalid element returns default",
			element:        4,
			expectedResult: 1,
			expectedFound:  false,
		},
		{
			name:           "first element",
			element:        1,
			expectedResult: 1,
			expectedFound:  true,
		},
		{
			name:           "last element",
			element:        9,
			expectedResult: 9,
			expectedFound:  true,
		},
		{
			name:           "negative element returns default",
			element:        -1,
			expectedResult: 1,
			expectedFound:  false,
		},
		{
			name:           "large element returns default",
			element:        100,
			expectedResult: 1,
			expectedFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := Sanitize(tt.element, allValues)
			if result != tt.expectedResult {
				t.Errorf("Expected result %d, got %d", tt.expectedResult, result)
			}
			if found != tt.expectedFound {
				t.Errorf("Expected found %v, got %v", tt.expectedFound, found)
			}
		})
	}
}

func TestSanitizeFloat64(t *testing.T) {
	// Test with float64 type
	allValues := func() ([]float64, float64) {
		return []float64{1.1, 2.2, 3.3}, 1.1
	}

	tests := []struct {
		name           string
		element        float64
		expectedResult float64
		expectedFound  bool
	}{
		{
			name:           "valid element",
			element:        2.2,
			expectedResult: 2.2,
			expectedFound:  true,
		},
		{
			name:           "zero element returns default",
			element:        0.0,
			expectedResult: 1.1,
			expectedFound:  true,
		},
		{
			name:           "invalid element returns default",
			element:        4.4,
			expectedResult: 1.1,
			expectedFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := Sanitize(tt.element, allValues)
			if result != tt.expectedResult {
				t.Errorf("Expected result %f, got %f", tt.expectedResult, result)
			}
			if found != tt.expectedFound {
				t.Errorf("Expected found %v, got %v", tt.expectedFound, found)
			}
		})
	}
}

func TestSanitizeEmptySlice(t *testing.T) {
	// Test with empty slice
	allValues := func() ([]string, string) {
		return []string{}, "default"
	}

	result, found := Sanitize("any", allValues)
	if result != "default" {
		t.Errorf("Expected result %q, got %q", "default", result)
	}
	if found {
		t.Errorf("Expected found to be false, got %v", found)
	}
}

func TestSanitizeEmptyDefault(t *testing.T) {
	// Test with empty default value
	allValues := func() ([]string, string) {
		return []string{"apple", "banana"}, ""
	}

	tests := []struct {
		name           string
		element        string
		expectedResult string
		expectedFound  bool
	}{
		{
			name:           "valid element",
			element:        "apple",
			expectedResult: "apple",
			expectedFound:  true,
		},
		{
			name:           "empty element with empty default",
			element:        "",
			expectedResult: "",
			expectedFound:  false,
		},
		{
			name:           "invalid element returns empty default",
			element:        "grape",
			expectedResult: "",
			expectedFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := Sanitize(tt.element, allValues)
			if result != tt.expectedResult {
				t.Errorf("Expected result %q, got %q", tt.expectedResult, result)
			}
			if found != tt.expectedFound {
				t.Errorf("Expected found %v, got %v", tt.expectedFound, found)
			}
		})
	}
}

func TestSanitizeWithOrder(t *testing.T) {
	// Test with Order enum type
	allValues := func() ([]Order, Order) {
		return []Order{OrderDefault, OrderAsc, OrderDesc}, OrderDefault
	}

	tests := []struct {
		name           string
		element        Order
		expectedResult Order
		expectedFound  bool
	}{
		{
			name:           "valid order",
			element:        OrderAsc,
			expectedResult: OrderAsc,
			expectedFound:  true,
		},
		{
			name:           "zero order returns default",
			element:        Order(0),
			expectedResult: OrderDefault,
			expectedFound:  true,
		},
		{
			name:           "invalid order returns default",
			element:        Order(999),
			expectedResult: OrderDefault,
			expectedFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := Sanitize(tt.element, allValues)
			if result != tt.expectedResult {
				t.Errorf("Expected result %v, got %v", tt.expectedResult, result)
			}
			if found != tt.expectedFound {
				t.Errorf("Expected found %v, got %v", tt.expectedFound, found)
			}
		})
	}
}

func TestToInterfaceSlice(t *testing.T) {
	// Test string slice
	stringSlice := []string{"a", "b", "c"}
	result := toInterfaceSlice(stringSlice)
	if len(result) != len(stringSlice) {
		t.Errorf("Expected length %d, got %d", len(stringSlice), len(result))
	}

	for i, v := range result {
		if v != stringSlice[i] {
			t.Errorf("Expected element %d to be %v, got %v", i, stringSlice[i], v)
		}
	}

	// Test int slice
	intSlice := []int{1, 2, 3}
	result = toInterfaceSlice(intSlice)

	if len(result) != len(intSlice) {
		t.Errorf("Expected length %d, got %d", len(intSlice), len(result))
	}

	for i, v := range result {
		if v != intSlice[i] {
			t.Errorf("Expected element %d to be %v, got %v", i, intSlice[i], v)
		}
	}

	// Test empty slice
	emptySlice := []string{}
	result = toInterfaceSlice(emptySlice)

	if len(result) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(result))
	}
}

func TestSortEnum(t *testing.T) {
	// Test string sorting
	stringSlice := []string{"zebra", "apple", "banana"}
	sorted := sortEnum(stringSlice)
	expected := []string{"apple", "banana", "zebra"}

	if len(sorted) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(sorted))
	}

	for i, v := range sorted {
		if v != expected[i] {
			t.Errorf("Expected element %d to be %q, got %q", i, expected[i], v)
		}
	}

	// Test int sorting
	intSlice := []int{3, 1, 4, 1, 5}
	sortedInt := sortEnum(intSlice)
	expectedInt := []int{1, 1, 3, 4, 5}

	if len(sortedInt) != len(expectedInt) {
		t.Errorf("Expected length %d, got %d", len(expectedInt), len(sortedInt))
	}

	for i, v := range sortedInt {
		if v != expectedInt[i] {
			t.Errorf("Expected element %d to be %d, got %d", i, expectedInt[i], v)
		}
	}

	// Test empty slice
	emptySlice := []string{}
	sortedEmpty := sortEnum(emptySlice)

	if len(sortedEmpty) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(sortedEmpty))
	}

	// Test single element
	singleSlice := []string{"single"}
	sortedSingle := sortEnum(singleSlice)

	if len(sortedSingle) != 1 || sortedSingle[0] != "single" {
		t.Errorf("Expected single element slice with 'single', got %v", sortedSingle)
	}
}

// Benchmark tests.
func BenchmarkSanitizeString(b *testing.B) {
	allValues := func() ([]string, string) {
		return []string{"apple", "banana", "cherry"}, "apple"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sanitize("banana", allValues)
	}
}

func BenchmarkSanitizeInt(b *testing.B) {
	allValues := func() ([]int, int) {
		return []int{1, 3, 5, 7, 9}, 1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sanitize(5, allValues)
	}
}

func BenchmarkToInterfaceSlice(b *testing.B) {
	stringSlice := []string{"a", "b", "c", "d", "e"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toInterfaceSlice(stringSlice)
	}
}

func BenchmarkSortEnum(b *testing.B) {
	stringSlice := []string{"zebra", "apple", "banana", "cherry", "date"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sortEnum(stringSlice)
	}
}
