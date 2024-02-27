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

package render

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/git"
)

func TestWriteErrorf(t *testing.T) {
	ctx := context.TODO()
	w := httptest.NewRecorder()

	e := usererror.New(500, "abc")
	UserError(ctx, w, e)

	if got, want := w.Code, 500; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	errjson := &usererror.Error{}
	if err := json.NewDecoder(w.Body).Decode(errjson); err != nil {
		t.Error(err)
	}
	if got, want := errjson.Message, e.Message; got != want {
		t.Errorf("Want error message %s, got %s", want, got)
	}
}

func TestWriteErrorCode(t *testing.T) {
	ctx := context.TODO()
	w := httptest.NewRecorder()

	UserError(ctx, w, usererror.Newf(418, "pc load letter %d", 1))

	if got, want := w.Code, 418; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	errjson := &usererror.Error{}
	if err := json.NewDecoder(w.Body).Decode(errjson); err != nil {
		t.Error(err)
	}
	if got, want := errjson.Message, "pc load letter 1"; got != want {
		t.Errorf("Want error message %s, got %s", want, got)
	}
}

func TestWriteNotFound(t *testing.T) {
	ctx := context.TODO()
	w := httptest.NewRecorder()

	NotFound(ctx, w)

	if got, want := w.Code, 404; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	errjson := &usererror.Error{}
	if err := json.NewDecoder(w.Body).Decode(errjson); err != nil {
		t.Error(err)
	}
	if got, want := errjson.Message, usererror.ErrNotFound.Message; got != want {
		t.Errorf("Want error message %s, got %s", want, got)
	}
}

func TestWriteUnauthorized(t *testing.T) {
	ctx := context.TODO()
	w := httptest.NewRecorder()

	Unauthorized(ctx, w)

	if got, want := w.Code, 401; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	errjson := &usererror.Error{}
	if err := json.NewDecoder(w.Body).Decode(errjson); err != nil {
		t.Error(err)
	}
	if got, want := errjson.Message, usererror.ErrUnauthorized.Message; got != want {
		t.Errorf("Want error message %s, got %s", want, got)
	}
}

func TestWriteForbidden(t *testing.T) {
	ctx := context.TODO()
	w := httptest.NewRecorder()

	Forbidden(ctx, w)

	if got, want := w.Code, 403; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	errjson := &usererror.Error{}
	if err := json.NewDecoder(w.Body).Decode(errjson); err != nil {
		t.Error(err)
	}
	if got, want := errjson.Message, usererror.ErrForbidden.Message; got != want {
		t.Errorf("Want error message %s, got %s", want, got)
	}
}

func TestWriteBadRequest(t *testing.T) {
	ctx := context.TODO()
	w := httptest.NewRecorder()

	BadRequest(ctx, w)

	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	errjson := &usererror.Error{}
	if err := json.NewDecoder(w.Body).Decode(errjson); err != nil {
		t.Error(err)
	}
	if got, want := errjson.Message, usererror.ErrBadRequest.Message; got != want {
		t.Errorf("Want error message %s, got %s", want, got)
	}
}

func TestWriteJSON(t *testing.T) {
	// without indent
	{
		w := httptest.NewRecorder()
		JSON(w, http.StatusTeapot, map[string]string{"hello": "world"})
		if got, want := w.Body.String(), "{\"hello\":\"world\"}\n"; got != want {
			t.Errorf("Want JSON body %q, got %q", want, got)
		}
		if got, want := w.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
			t.Errorf("Want Content-Type %q, got %q", want, got)
		}
		if got, want := w.Code, http.StatusTeapot; got != want {
			t.Errorf("Want status code %d, got %d", want, got)
		}
	}
	// with indent
	{
		indent = true
		defer func() {
			indent = false
		}()
		w := httptest.NewRecorder()
		JSON(w, http.StatusTeapot, map[string]string{"hello": "world"})
		if got, want := w.Body.String(), "{\n  \"hello\": \"world\"\n}\n"; got != want {
			t.Errorf("Want JSON body %q, got %q", want, got)
		}
	}
}

func TestJSONArrayDynamic(t *testing.T) {
	noctx := context.Background()
	type mock struct {
		ID int `json:"id"`
	}
	type args[T comparable] struct {
		ctx context.Context
		f   func(ch chan<- *mock, cherr chan<- error)
	}
	type testCase[T comparable] struct {
		name    string
		args    args[T]
		len     int
		wantErr bool
	}

	tests := []testCase[*mock]{
		{
			name: "happy path",
			args: args[*mock]{
				ctx: noctx,
				f: func(ch chan<- *mock, _ chan<- error) {
					defer close(ch)
					ch <- &mock{ID: 1}
				},
			},
			len:     1,
			wantErr: false,
		},
		{
			name: "empty array response",
			args: args[*mock]{
				ctx: noctx,
				f: func(ch chan<- *mock, _ chan<- error) {
					close(ch)
				},
			},
			len:     0,
			wantErr: false,
		},
		{
			name: "error at beginning of the stream",
			args: args[*mock]{
				ctx: noctx,
				f: func(ch chan<- *mock, cherr chan<- error) {
					defer close(ch)
					defer close(cherr)
					cherr <- errors.New("error writing to the response writer")
				},
			},
			len:     0,
			wantErr: true,
		},
		{
			name: "error while streaming",
			args: args[*mock]{
				ctx: noctx,
				f: func(ch chan<- *mock, cherr chan<- error) {
					defer close(ch)
					defer close(cherr)
					ch <- &mock{ID: 1}
					ch <- &mock{ID: 2}
					cherr <- errors.New("error writing to the response writer")
				},
			},
			len:     2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan *mock)
			cherr := make(chan error, 1)

			stream := git.NewStreamReader(ch, cherr)
			go func() {
				tt.args.f(ch, cherr)
			}()

			w := httptest.NewRecorder()

			JSONArrayDynamic[*mock](tt.args.ctx, w, stream)

			ct := w.Header().Get("Content-Type")
			if ct != "application/json; charset=utf-8" {
				t.Errorf("Content type should be application/json, got: %v", ct)
				return
			}

			if tt.wantErr {
				if w.Code != 500 {
					t.Errorf("stream error code should be 500, got: %v", w.Code)
				}
				return
			}

			var m []mock
			err := json.NewDecoder(w.Body).Decode(&m)
			if err != nil {
				t.Errorf("error should be nil, got: %v", err)
				return
			}

			if len(m) != tt.len {
				t.Errorf("json array length should be %d, got: %v", tt.len, len(m))
				return
			}
		})
	}
}
