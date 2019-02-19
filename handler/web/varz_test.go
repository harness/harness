// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package web

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/go-scm/scm"
	"github.com/google/go-cmp/cmp"
)

func TestHandleVarz(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	client := new(scm.Client)
	client.BaseURL, _ = url.Parse("https://github.com")
	client.SetRate(scm.Rate{
		Limit:     5000,
		Remaining: 875,
		Reset:     1523640878,
	})

	license := &core.License{
		Kind:  core.LicenseStandard,
		Repos: 50,
		Users: 100,
	}
	HandleVarz(client, license).ServeHTTP(w, r)

	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := &varz{}, mockVarz
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

var mockVarz = &varz{
	SCM: &scmInfo{
		URL: "https://github.com",
		Rate: &rateInfo{
			Limit:     5000,
			Remaining: 875,
			Reset:     1523640878,
		},
	},
	License: &licenseInfo{
		Kind:       "standard",
		Seats:      100,
		SeatsUsed:  0,
		SeatsAvail: 0,
		Repos:      50,
		ReposUsed:  0,
		ReposAvail: 0,
	},
}
