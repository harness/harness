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

package lock

import "testing"

const myKey = "mykey"

func TestFormatKey(t *testing.T) {
	t.Run("format with all parts", func(t *testing.T) {
		result := formatKey("myapp", "mynamespace", myKey)
		expected := "myapp:mynamespace:mykey"
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("format with empty app", func(t *testing.T) {
		result := formatKey("", "mynamespace", myKey)
		expected := ":mynamespace:mykey"
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("format with empty namespace", func(t *testing.T) {
		result := formatKey("myapp", "", myKey)
		expected := "myapp::mykey"
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("format with empty key", func(t *testing.T) {
		result := formatKey("myapp", "mynamespace", "")
		expected := "myapp:mynamespace:"
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("format with all empty", func(t *testing.T) {
		result := formatKey("", "", "")
		expected := "::"
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("format with special characters", func(t *testing.T) {
		result := formatKey("app-1", "ns_2", "key.3")
		expected := "app-1:ns_2:key.3"
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})
}

func TestSplitKey(t *testing.T) {
	t.Run("split valid key with three parts", func(t *testing.T) {
		namespace, key := SplitKey("myapp:mynamespace:mykey")
		if namespace != "mynamespace" {
			t.Errorf("expected namespace to be 'mynamespace', got '%s'", namespace)
		}
		if key != myKey {
			t.Errorf("expected key to be 'mykey', got '%s'", key)
		}
	})

	t.Run("split key with more than three parts", func(t *testing.T) {
		namespace, key := SplitKey("myapp:mynamespace:mykey:extra")
		if namespace != "mynamespace" {
			t.Errorf("expected namespace to be 'mynamespace', got '%s'", namespace)
		}
		// SplitKey only takes the third part, not everything after
		if key != myKey {
			t.Errorf("expected key to be 'mykey', got '%s'", key)
		}
	})

	t.Run("split key with two parts", func(t *testing.T) {
		namespace, key := SplitKey("myapp:mynamespace")
		if namespace != "" {
			t.Errorf("expected namespace to be empty, got '%s'", namespace)
		}
		if key != "myapp:mynamespace" {
			t.Errorf("expected key to be 'myapp:mynamespace', got '%s'", key)
		}
	})

	t.Run("split key with one part", func(t *testing.T) {
		namespace, key := SplitKey(myKey)
		if namespace != "" {
			t.Errorf("expected namespace to be empty, got '%s'", namespace)
		}
		if key != myKey {
			t.Errorf("expected key to be 'mykey', got '%s'", key)
		}
	})

	t.Run("split empty key", func(t *testing.T) {
		namespace, key := SplitKey("")
		if namespace != "" {
			t.Errorf("expected namespace to be empty, got '%s'", namespace)
		}
		if key != "" {
			t.Errorf("expected key to be empty, got '%s'", key)
		}
	})

	t.Run("split key with empty parts", func(t *testing.T) {
		namespace, key := SplitKey("myapp::mykey")
		if namespace != "" {
			t.Errorf("expected namespace to be empty, got '%s'", namespace)
		}
		if key != myKey {
			t.Errorf("expected key to be 'mykey', got '%s'", key)
		}
	})
}

func TestFormatAndSplitKey_RoundTrip(t *testing.T) {
	testCases := []struct {
		app       string
		namespace string
		key       string
	}{
		{"myapp", "mynamespace", myKey},
		{"app1", "ns1", "key1"},
		{"", "ns", "key"},
		{"app", "", "key"},
		{"app", "ns", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.app+":"+tc.namespace+":"+tc.key, func(t *testing.T) {
			formatted := formatKey(tc.app, tc.namespace, tc.key)
			ns, k := SplitKey(formatted)

			if ns != tc.namespace {
				t.Errorf("namespace mismatch: expected '%s', got '%s'", tc.namespace, ns)
			}
			if k != tc.key {
				t.Errorf("key mismatch: expected '%s', got '%s'", tc.key, k)
			}
		})
	}
}
