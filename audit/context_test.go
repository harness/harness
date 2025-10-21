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

package audit

import (
	"context"
	"testing"
)

func TestGetRealIP(t *testing.T) {
	t.Run("returns IP when present", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), realIPKey, "192.168.1.1")
		ip := GetRealIP(ctx)
		if ip != "192.168.1.1" {
			t.Errorf("expected IP to be '192.168.1.1', got '%s'", ip)
		}
	})

	t.Run("returns empty string when not present", func(t *testing.T) {
		ctx := context.Background()
		ip := GetRealIP(ctx)
		if ip != "" {
			t.Errorf("expected empty string, got '%s'", ip)
		}
	})

	t.Run("returns empty string when wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), realIPKey, 12345)
		ip := GetRealIP(ctx)
		if ip != "" {
			t.Errorf("expected empty string, got '%s'", ip)
		}
	})

	t.Run("handles IPv6 address", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), realIPKey, "2001:0db8:85a3:0000:0000:8a2e:0370:7334")
		ip := GetRealIP(ctx)
		if ip != "2001:0db8:85a3:0000:0000:8a2e:0370:7334" {
			t.Errorf("expected IPv6 address, got '%s'", ip)
		}
	})
}

func TestGetPath(t *testing.T) {
	t.Run("returns path when present", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), pathKey, "/api/v1/users")
		path := GetPath(ctx)
		if path != "/api/v1/users" {
			t.Errorf("expected path to be '/api/v1/users', got '%s'", path)
		}
	})

	t.Run("returns empty string when not present", func(t *testing.T) {
		ctx := context.Background()
		path := GetPath(ctx)
		if path != "" {
			t.Errorf("expected empty string, got '%s'", path)
		}
	})

	t.Run("returns empty string when wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), pathKey, 12345)
		path := GetPath(ctx)
		if path != "" {
			t.Errorf("expected empty string, got '%s'", path)
		}
	})

	t.Run("handles empty path", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), pathKey, "")
		path := GetPath(ctx)
		if path != "" {
			t.Errorf("expected empty string, got '%s'", path)
		}
	})
}

func TestGetRequestID(t *testing.T) {
	t.Run("returns request ID when present", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), requestID, "req-12345")
		id := GetRequestID(ctx)
		if id != "req-12345" {
			t.Errorf("expected request ID to be 'req-12345', got '%s'", id)
		}
	})

	t.Run("returns empty string when not present", func(t *testing.T) {
		ctx := context.Background()
		id := GetRequestID(ctx)
		if id != "" {
			t.Errorf("expected empty string, got '%s'", id)
		}
	})

	t.Run("returns empty string when wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), requestID, 12345)
		id := GetRequestID(ctx)
		if id != "" {
			t.Errorf("expected empty string, got '%s'", id)
		}
	})

	t.Run("handles UUID format", func(t *testing.T) {
		uuid := "550e8400-e29b-41d4-a716-446655440000"
		ctx := context.WithValue(context.Background(), requestID, uuid)
		id := GetRequestID(ctx)
		if id != uuid {
			t.Errorf("expected request ID to be '%s', got '%s'", uuid, id)
		}
	})
}

func TestGetRequestMethod(t *testing.T) {
	t.Run("returns method when present", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), requestMethod, "GET")
		method := GetRequestMethod(ctx)
		if method != "GET" {
			t.Errorf("expected method to be 'GET', got '%s'", method)
		}
	})

	t.Run("returns empty string when not present", func(t *testing.T) {
		ctx := context.Background()
		method := GetRequestMethod(ctx)
		if method != "" {
			t.Errorf("expected empty string, got '%s'", method)
		}
	})

	t.Run("returns empty string when wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), requestMethod, 12345)
		method := GetRequestMethod(ctx)
		if method != "" {
			t.Errorf("expected empty string, got '%s'", method)
		}
	})

	t.Run("handles various HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
		for _, m := range methods {
			ctx := context.WithValue(context.Background(), requestMethod, m)
			method := GetRequestMethod(ctx)
			if method != m {
				t.Errorf("expected method to be '%s', got '%s'", m, method)
			}
		}
	})
}
