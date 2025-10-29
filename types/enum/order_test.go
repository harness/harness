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

import "testing"

func TestParseOrder(t *testing.T) {
	tests := []struct {
		text string
		want Order
	}{
		{"asc", OrderAsc},
		{"Asc", OrderAsc},
		{"ASC", OrderAsc},
		{"ascending", OrderAsc},
		{"Ascending", OrderAsc},
		{"ASCENDING", OrderAsc},
		{"desc", OrderDesc},
		{"Desc", OrderDesc},
		{"DESC", OrderDesc},
		{"descending", OrderDesc},
		{"Descending", OrderDesc},
		{"DESCENDING", OrderDesc},
		{"", OrderDefault},
		{"invalid", OrderDefault},
		{"random", OrderDefault},
		{"123", OrderDefault},
		{"asc ", OrderDefault},    // trailing space
		{" asc", OrderDefault},    // leading space
		{"asc\n", OrderDefault},   // newline
		{"asc\t", OrderDefault},   // tab
		{"ascend", OrderDefault},  // partial match
		{"descend", OrderDefault}, // partial match
		{"ascc", OrderDefault},    // typo
		{"descc", OrderDefault},   // typo
		{"null", OrderDefault},
		{"undefined", OrderDefault},
	}

	for _, test := range tests {
		t.Run(test.text, func(t *testing.T) {
			got, want := ParseOrder(test.text), test.want
			if got != want {
				t.Errorf("Want order %q parsed as %q, got %q", test.text, want, got)
			}
		})
	}
}

func TestOrderString(t *testing.T) {
	tests := []struct {
		order Order
		want  string
	}{
		{OrderDefault, "desc"}, // OrderDefault returns desc
		{OrderAsc, "asc"},
		{OrderDesc, "desc"},
		{Order(999), "undefined"}, // invalid order value
		{Order(-1), "undefined"},  // negative order value
		{Order(100), "undefined"}, // large order value
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			got := test.order.String()
			if got != test.want {
				t.Errorf("Want order %v as string %q, got %q", test.order, test.want, got)
			}
		})
	}
}

func TestOrderConstants(t *testing.T) {
	// Test that the constants have expected values
	if OrderDefault != 0 {
		t.Errorf("Expected OrderDefault to be 0, got %d", OrderDefault)
	}
	if OrderAsc != 1 {
		t.Errorf("Expected OrderAsc to be 1, got %d", OrderAsc)
	}
	if OrderDesc != 2 {
		t.Errorf("Expected OrderDesc to be 2, got %d", OrderDesc)
	}
}

func TestOrderStringRoundTrip(t *testing.T) {
	// Test that parsing the string representation gives back the original order
	orders := []Order{OrderDefault, OrderAsc, OrderDesc}

	for _, order := range orders {
		t.Run(order.String(), func(t *testing.T) {
			str := order.String()
			parsed := ParseOrder(str)

			// Note: OrderDefault.String() returns "desc", so parsing it gives OrderDesc
			// This is expected behavior based on the implementation
			if order == OrderDefault {
				if parsed != OrderDesc {
					t.Errorf("Expected parsing OrderDefault string to give OrderDesc, got %v", parsed)
				}
			} else {
				if parsed != order {
					t.Errorf("Expected parsing %v string to give %v, got %v", order, order, parsed)
				}
			}
		})
	}
}

func TestOrderComparison(t *testing.T) {
	// Test that orders can be compared
	if OrderDefault >= OrderAsc {
		t.Error("Expected OrderDefault < OrderAsc")
	}
	if OrderAsc >= OrderDesc {
		t.Error("Expected OrderAsc < OrderDesc")
	}
	if OrderDefault >= OrderDesc {
		t.Error("Expected OrderDefault < OrderDesc")
	}
}

func TestOrderType(t *testing.T) {
	// Test that Order is the correct type
	var o Order
	if o != OrderDefault {
		t.Errorf("Expected zero value of Order to be OrderDefault, got %v", o)
	}

	// Test type conversion
	o = Order(1)
	if o != OrderAsc {
		t.Errorf("Expected Order(1) to be OrderAsc, got %v", o)
	}
}

// Benchmark tests.
func BenchmarkParseOrder(b *testing.B) {
	for b.Loop() {
		ParseOrder("asc")
	}
}

func BenchmarkParseOrderDesc(b *testing.B) {
	for b.Loop() {
		ParseOrder("desc")
	}
}

func BenchmarkParseOrderInvalid(b *testing.B) {
	for b.Loop() {
		ParseOrder("invalid")
	}
}

func BenchmarkOrderString(b *testing.B) {
	for b.Loop() {
		_ = OrderAsc.String()
	}
}

func BenchmarkOrderStringDesc(b *testing.B) {
	for b.Loop() {
		_ = OrderDesc.String()
	}
}

func BenchmarkOrderStringDefault(b *testing.B) {
	for b.Loop() {
		_ = OrderDefault.String()
	}
}
