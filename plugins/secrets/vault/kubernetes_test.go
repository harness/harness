package vault

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetKubernetesToken(t *testing.T) {
	fakeRole := "fakeRole"
	fakeMountPoint := "kubernetes"
	fakeJwtFile := "fixtures/fakeJwt"
	b, _ := ioutil.ReadFile(fakeJwtFile)
	fakeJwt := string(b)
	fakeClientToken := "fakeClientToken"
	fakeLeaseSeconds := 86400
	fakeLeaseDuration := time.Duration(fakeLeaseSeconds) * time.Second
	fakeResp := fmt.Sprintf("{\"auth\": {\"client_token\": \"%s\", \"lease_duration\": %d}}", fakeClientToken, fakeLeaseSeconds)
	expectedPath := "/v1/auth/kubernetes/login"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got '%s'", r.Method)
		}
		if r.URL.EscapedPath() != expectedPath {
			t.Errorf("Expected request to '%s', got '%s'", expectedPath, r.URL.EscapedPath())
		}

		var postdata struct {
			Jwt  string
			Role string
		}
		err := json.NewDecoder(r.Body).Decode(&postdata)
		if err != nil {
			t.Errorf("Encountered error parsing request JSON:  %s", err)
		}

		jwt := postdata.Jwt

		if jwt != fakeJwt {
			t.Errorf("Expected request to have jwt with value '%s', got: '%s'", fakeJwt, jwt)
		}
		role := postdata.Role
		if role != fakeRole {
			t.Errorf("Expected request to have role with value '%s', got: '%s'", fakeRole, role)
		}

		fmt.Fprintf(w, fakeResp)
	}))
	defer ts.Close()

	url := ts.URL
	token, ttl, err := getKubernetesToken(url, fakeRole, fakeMountPoint, fakeJwtFile)
	if err != nil {
		t.Errorf("getKubernetesToken returned an error: %s", err)
	}

	if token != fakeClientToken {
		t.Errorf("Expected returned token to have value '%s', got: '%s'", fakeClientToken, token)
	}
	if ttl != fakeLeaseDuration {
		t.Errorf("Expected TTL to have value '%s', got: '%s'", fakeLeaseDuration.Seconds(), ttl.Seconds())
	}
}
