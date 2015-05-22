package logrus_airbrake

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/tobi/airbrake-go"
)

type notice struct {
	Error struct {
		Message string `xml:"message"`
	} `xml:"error"`
}

func TestNoticeReceived(t *testing.T) {
	msg := make(chan string, 1)
	expectedMsg := "foo"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var notice notice
		if err := xml.NewDecoder(r.Body).Decode(&notice); err != nil {
			t.Error(err)
		}
		r.Body.Close()

		msg <- notice.Error.Message
	}))
	defer ts.Close()

	hook := &AirbrakeHook{}

	airbrake.Environment = "production"
	airbrake.Endpoint = ts.URL
	airbrake.ApiKey = "foo"

	log := logrus.New()
	log.Hooks.Add(hook)

	log.WithFields(logrus.Fields{
		"error": errors.New(expectedMsg),
	}).Error("Airbrake will not see this string")

	select {
	case received := <-msg:
		if received != expectedMsg {
			t.Errorf("Unexpected message received: %s", received)
		}
	case <-time.After(time.Second):
		t.Error("Timed out; no notice received by Airbrake API")
	}
}
