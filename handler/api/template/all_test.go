// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package template

//func TestHandleAll(t *testing.T) {
//	controller := gomock.NewController(t)
//	defer controller.Finish()
//
//	secrets := mock.NewMockGlobalSecretStore(controller)
//	secrets.EXPECT().ListAll(gomock.Any()).Return(dummySecretList, nil)
//
//	w := httptest.NewRecorder()
//	r := httptest.NewRequest("GET", "/", nil)
//
//	HandleAll(secrets).ServeHTTP(w, r)
//	if got, want := w.Code, http.StatusOK; want != got {
//		t.Errorf("Want response code %d, got %d", want, got)
//	}
//
//	got, want := []*core.Secret{}, dummySecretListScrubbed
//	json.NewDecoder(w.Body).Decode(&got)
//	if diff := cmp.Diff(got, want); len(diff) != 0 {
//		t.Errorf(diff)
//	}
//}
