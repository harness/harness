// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package template

import (
	"github.com/drone/drone/mock"
	"github.com/golang/mock/gomock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAll(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	templates := mock.NewMockTemplateStore(controller)
	templates.EXPECT().ListAll(gomock.Any()).Return(dummyTemplateList, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	HandleAll(templates).ServeHTTP(w, r)
	if got, want := w.Code, http.StatusOK; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}
