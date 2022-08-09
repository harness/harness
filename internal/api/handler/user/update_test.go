// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/bradrydzewski/my-app/internal/api/render"
	"github.com/bradrydzewski/my-app/internal/api/request"
	"github.com/bradrydzewski/my-app/mocks"
	"github.com/bradrydzewski/my-app/types"
	"golang.org/x/crypto/bcrypt"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/gotidy/ptr"
)

// mock bcrypt has function returns an error
// when attepting to has the password.
func bcryptHashErrror(password []byte, cost int) ([]byte, error) {
	return nil, bcrypt.ErrHashTooShort
}

// mock bcrypt has function returns a deterministic
// hash value for testing purposes.
func bcryptHashStatic(password []byte, cost int) ([]byte, error) {
	return []byte("$2a$10$onMfkmQZtlkOfnZJe7GaiesbPBbXcyB53KyFKllWq829mxlhNoJSi"), nil
}

func TestUpdate(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	hashPassword = bcryptHashStatic
	defer func() {
		hashPassword = bcrypt.GenerateFromPassword
	}()

	userInput := &types.UserInput{
		Username: ptr.String("octocat@google.com"),
		Password: ptr.String("password"),
		Company:  ptr.String("google"),
	}
	before := &types.User{
		Email:    "octocat@google.com",
		Password: "acme",
	}

	users := mocks.NewMockUserStore(controller)
	users.EXPECT().Update(gomock.Any(), before)

	in := new(bytes.Buffer)
	json.NewEncoder(in).Encode(userInput)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/v1/user", in)
	r = r.WithContext(
		request.WithUser(r.Context(), before),
	)

	HandleUpdate(users)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	if got, want := before.Email, "octocat@google.com"; got != want {
		t.Errorf("Want user email %v, got %v", want, got)
	}
	if got, want := before.Password, "$2a$10$onMfkmQZtlkOfnZJe7GaiesbPBbXcyB53KyFKllWq829mxlhNoJSi"; got != want {
		t.Errorf("Want user password %v, got %v", want, got)
	}

	after := &types.User{
		Email:   "octocat@google.com",
		Company: "google",
		// Password hash is not exposecd to JSON
	}
	got, want := new(types.User), after
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// the purpose of this unit test is to verify that a
// failure to hash the password will return an internal
// server error.
func TestUpdate_HashError(t *testing.T) {
	hashPassword = bcryptHashErrror
	defer func() {
		hashPassword = bcrypt.GenerateFromPassword
	}()

	controller := gomock.NewController(t)
	defer controller.Finish()

	userInput := &types.UserInput{
		Username: ptr.String("octocat@github.com"),
		Password: ptr.String("password"),
	}
	user := &types.User{
		Email: "octocat@github.com",
	}

	in := new(bytes.Buffer)
	json.NewEncoder(in).Encode(userInput)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/v1/user", in)
	r = r.WithContext(
		request.WithUser(r.Context(), user),
	)

	HandleUpdate(nil)(w, r)
	if got, want := w.Code, 500; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(render.Error), &render.Error{Message: bcrypt.ErrHashTooShort.Error()}
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// the purpose of this unit test is to verify that an invalid
// (in this case missing) request body will result in a bad
// request error returned to the client.
func TestUpdate_BadRequest(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &types.User{
		ID:    1,
		Email: "octocat@github.com",
	}

	in := new(bytes.Buffer)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/v1/user", in)
	r = r.WithContext(
		request.WithUser(r.Context(), mockUser),
	)

	HandleUpdate(nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(render.Error), &render.Error{Message: "EOF"}
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// the purpose of this unit test is to verify that an error
// updating the database will result in an internal server
// error returned to the client.
func TestUpdate_ServerError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	userInput := &types.UserInput{
		Username: ptr.String("octocat@github.com"),
	}
	user := &types.User{
		Email: "octocat@github.com",
	}

	users := mocks.NewMockUserStore(controller)
	users.EXPECT().Update(gomock.Any(), user).Return(render.ErrNotFound)

	in := new(bytes.Buffer)
	json.NewEncoder(in).Encode(userInput)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/v1/user", in)
	r = r.WithContext(
		request.WithUser(r.Context(), user),
	)

	HandleUpdate(users)(w, r)
	if got, want := w.Code, 500; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(render.Error), render.ErrNotFound
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}
