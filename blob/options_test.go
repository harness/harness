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

package blob

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestSignURLConfig(t *testing.T) {
	tests := []struct {
		name   string
		config SignURLConfig
	}{
		{
			name:   "empty config",
			config: SignURLConfig{},
		},
		{
			name: "full config",
			config: SignURLConfig{
				Method:          "POST",
				ContentType:     "application/json",
				Headers:         []string{"Authorization", "X-Custom-Header"},
				QueryParameters: url.Values{"param1": []string{"value1"}, "param2": []string{"value2"}},
				Insecure:        true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := test.config

			if config.Method != test.config.Method {
				t.Errorf("expected method %q, got %q", test.config.Method, config.Method)
			}

			if config.ContentType != test.config.ContentType {
				t.Errorf("expected content type %q, got %q", test.config.ContentType, config.ContentType)
			}

			if !reflect.DeepEqual(config.Headers, test.config.Headers) {
				t.Errorf("expected headers %v, got %v", test.config.Headers, config.Headers)
			}

			if !reflect.DeepEqual(config.QueryParameters, test.config.QueryParameters) {
				t.Errorf("expected query parameters %v, got %v", test.config.QueryParameters, config.QueryParameters)
			}

			if config.Insecure != test.config.Insecure {
				t.Errorf("expected insecure %v, got %v", test.config.Insecure, config.Insecure)
			}
		})
	}
}

func TestSignWithMethod(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected string
	}{
		{
			name:     "GET method",
			method:   "GET",
			expected: "GET",
		},
		{
			name:     "POST method",
			method:   "POST",
			expected: "POST",
		},
		{
			name:     "PUT method",
			method:   "PUT",
			expected: "PUT",
		},
		{
			name:     "DELETE method",
			method:   "DELETE",
			expected: "DELETE",
		},
		{
			name:     "empty method",
			method:   "",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &SignURLConfig{}
			option := SignWithMethod(test.method)
			option.Apply(config)

			if config.Method != test.expected {
				t.Errorf("expected method %q, got %q", test.expected, config.Method)
			}
		})
	}
}

func TestSignWithContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    string
	}{
		{
			name:        "JSON content type",
			contentType: "application/json",
			expected:    "application/json",
		},
		{
			name:        "XML content type",
			contentType: "application/xml",
			expected:    "application/xml",
		},
		{
			name:        "plain text content type",
			contentType: "text/plain",
			expected:    "text/plain",
		},
		{
			name:        "empty content type",
			contentType: "",
			expected:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &SignURLConfig{}
			option := SignWithContentType(test.contentType)
			option.Apply(config)

			if config.ContentType != test.expected {
				t.Errorf("expected content type %q, got %q", test.expected, config.ContentType)
			}
		})
	}
}

func TestSignWithHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  []string
		expected []string
	}{
		{
			name:     "single header",
			headers:  []string{"Authorization"},
			expected: []string{"Authorization"},
		},
		{
			name:     "multiple headers",
			headers:  []string{"Authorization", "X-Custom-Header", "Content-Type"},
			expected: []string{"Authorization", "X-Custom-Header", "Content-Type"},
		},
		{
			name:     "empty headers",
			headers:  []string{},
			expected: []string{},
		},
		{
			name:     "nil headers",
			headers:  nil,
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &SignURLConfig{}
			option := SignWithHeaders(test.headers)
			option.Apply(config)

			if !reflect.DeepEqual(config.Headers, test.expected) {
				t.Errorf("expected headers %v, got %v", test.expected, config.Headers)
			}
		})
	}
}

func TestSignWithQueryParameters(t *testing.T) {
	tests := []struct {
		name     string
		params   url.Values
		expected url.Values
	}{
		{
			name:     "single parameter",
			params:   url.Values{"key": []string{"value"}},
			expected: url.Values{"key": []string{"value"}},
		},
		{
			name:     "multiple parameters",
			params:   url.Values{"key1": []string{"value1"}, "key2": []string{"value2", "value3"}},
			expected: url.Values{"key1": []string{"value1"}, "key2": []string{"value2", "value3"}},
		},
		{
			name:     "empty parameters",
			params:   url.Values{},
			expected: url.Values{},
		},
		{
			name:     "nil parameters",
			params:   nil,
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &SignURLConfig{}
			option := SignWithQueryParameters(test.params)
			option.Apply(config)

			if !reflect.DeepEqual(config.QueryParameters, test.expected) {
				t.Errorf("expected query parameters %v, got %v", test.expected, config.QueryParameters)
			}
		})
	}
}

func TestSignWithInsecure(t *testing.T) {
	tests := []struct {
		name     string
		insecure bool
		expected bool
	}{
		{
			name:     "insecure true",
			insecure: true,
			expected: true,
		},
		{
			name:     "insecure false",
			insecure: false,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &SignURLConfig{}
			option := SignWithInsecure(test.insecure)
			option.Apply(config)

			if config.Insecure != test.expected {
				t.Errorf("expected insecure %v, got %v", test.expected, config.Insecure)
			}
		})
	}
}

func TestSignedURLConfigFunc(t *testing.T) {
	// Test that SignedURLConfigFunc implements SignURLOption interface
	var _ SignURLOption = SignedURLConfigFunc(func(_ *SignURLConfig) {})

	// Test custom function
	customFunc := SignedURLConfigFunc(func(opts *SignURLConfig) {
		opts.Method = "CUSTOM"
		opts.ContentType = "custom/type"
		opts.Insecure = true
	})

	config := &SignURLConfig{}
	customFunc.Apply(config)

	if config.Method != "CUSTOM" {
		t.Errorf("expected method 'CUSTOM', got %q", config.Method)
	}

	if config.ContentType != "custom/type" {
		t.Errorf("expected content type 'custom/type', got %q", config.ContentType)
	}

	if !config.Insecure {
		t.Error("expected insecure to be true")
	}
}

func TestMultipleOptions(t *testing.T) {
	config := &SignURLConfig{}

	// Apply multiple options
	options := []SignURLOption{
		SignWithMethod("POST"),
		SignWithContentType("application/json"),
		SignWithHeaders([]string{"Authorization", "X-Custom"}),
		SignWithQueryParameters(url.Values{"test": []string{"value"}}),
		SignWithInsecure(true),
	}

	for _, option := range options {
		option.Apply(config)
	}

	// Verify all options were applied
	if config.Method != http.MethodPost {
		t.Errorf("expected method 'POST', got %q", config.Method)
	}

	if config.ContentType != "application/json" {
		t.Errorf("expected content type 'application/json', got %q", config.ContentType)
	}

	expectedHeaders := []string{"Authorization", "X-Custom"}
	if !reflect.DeepEqual(config.Headers, expectedHeaders) {
		t.Errorf("expected headers %v, got %v", expectedHeaders, config.Headers)
	}

	expectedParams := url.Values{"test": []string{"value"}}
	if !reflect.DeepEqual(config.QueryParameters, expectedParams) {
		t.Errorf("expected query parameters %v, got %v", expectedParams, config.QueryParameters)
	}

	if !config.Insecure {
		t.Error("expected insecure to be true")
	}
}
