// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package logger

import (
	"net/http/httptest"
	"testing"
)

func TestMiddleware(t *testing.T) {
	t.Skip()
}

func TestMiddleware_GenerateRequestID(t *testing.T) {
	t.Skip()
}

func TestAuthType(t *testing.T) {
	cookieRequest := httptest.NewRequest("GET", "/", nil)
	if authType(cookieRequest) != "cookie" {
		t.Error("authtype is not cookie")
	}

	headerRequest := httptest.NewRequest("GET", "/", nil)
	headerRequest.Header.Add("Authorization", "test")
	if authType(headerRequest) != "token" {
		t.Error("authtype is not token")
	}

	formRequest := httptest.NewRequest("GET", "/?access_token=test", nil)
	if authType(formRequest) != "token" {
		t.Error("authtype is not token")
	}
}
