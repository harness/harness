// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/bradrydzewski/my-app/internal/api/request"
	"github.com/bradrydzewski/my-app/types"
	"github.com/dgrijalva/jwt-go"

	"github.com/golang/mock/gomock"
)

func TestToken(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &types.User{
		ID:    1,
		Email: "octocat@github.com",
		Salt:  "12345",
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		request.WithUser(r.Context(), mockUser),
	)

	HandleToken(nil)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	result := &types.Token{}
	json.NewDecoder(w.Body).Decode(&result)

	_, err := jwt.Parse(result.Value, func(token *jwt.Token) (interface{}, error) {
		return []byte(mockUser.Salt), nil
	})
	if err != nil {
		t.Error(err)
	}
}
