// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package metric

import (
	"net/http/httptest"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/golang/mock/gomock"
)

func TestHandleMetrics(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	mockUser := &core.User{Admin: false, Machine: true}
	session := mock.NewMockSession(controller)
	session.EXPECT().Get(r).Return(mockUser, nil)

	NewServer(session, false).ServeHTTP(w, r)
	if got, want := w.Code, 200; got != want {
		t.Errorf("Want status code %d, got %d", want, got)
	}

	if got, want := w.HeaderMap.Get("Content-Type"), "text/plain; version=0.0.4; charset=utf-8"; got != want {
		t.Errorf("Want prometheus header %q, got %q", want, got)
	}
}

func TestHandleMetrics_NoSession(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	session := mock.NewMockSession(controller)
	session.EXPECT().Get(r).Return(nil, nil)

	NewServer(session, false).ServeHTTP(w, r)

	if got, want := w.Code, 401; got != want {
		t.Errorf("Want status code %d, got %d", want, got)
	}
}

func TestHandleMetrics_NoSessionButAnonymousAccessEnabled(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	session := mock.NewMockSession(controller)
	session.EXPECT().Get(r).Return(nil, nil)

	NewServer(session, true).ServeHTTP(w, r)

	if got, want := w.Code, 200; got != want {
		t.Errorf("Want status code %d, got %d", want, got)
	}
}

func TestHandleMetrics_AccessDenied(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	mockUser := &core.User{Admin: false, Machine: false}
	session := mock.NewMockSession(controller)
	session.EXPECT().Get(r).Return(mockUser, nil)

	NewServer(session, false).ServeHTTP(w, r)
	if got, want := w.Code, 403; got != want {
		t.Errorf("Want status code %d, got %d", want, got)
	}
}
