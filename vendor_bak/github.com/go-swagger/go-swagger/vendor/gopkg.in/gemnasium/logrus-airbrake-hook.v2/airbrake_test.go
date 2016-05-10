package airbrake

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"gopkg.in/airbrake/gobrake.v2"
)

type customErr struct {
	msg string
}

func (e *customErr) Error() string {
	return e.msg
}

const (
	testAPIKey    = "abcxyz"
	testEnv       = "development"
	expectedClass = "*airbrake.customErr"
	expectedMsg   = "foo"
	unintendedMsg = "Airbrake will not see this string"
)

var (
	noticeChan = make(chan *gobrake.Notice, 1)
)

// TestLogEntryMessageReceived checks if invoking Logrus' log.Error
// method causes an XML payload containing the log entry message is received
// by a HTTP server emulating an Airbrake-compatible endpoint.
func TestLogEntryMessageReceived(t *testing.T) {
	log := logrus.New()
	hook := newTestHook()
	log.Hooks.Add(hook)

	log.Error(expectedMsg)

	select {
	case received := <-noticeChan:
		receivedErr := received.Errors[0]
		if receivedErr.Message != expectedMsg {
			t.Errorf("Unexpected message received: %s", receivedErr.Message)
		}
	case <-time.After(time.Second):
		t.Error("Timed out; no notice received by Airbrake API")
	}
}

// TestLogEntryMessageReceived confirms that, when passing an error type using
// logrus.Fields, a HTTP server emulating an Airbrake endpoint receives the
// error message returned by the Error() method on the error interface
// rather than the logrus.Entry.Message string.
func TestLogEntryWithErrorReceived(t *testing.T) {
	log := logrus.New()
	hook := newTestHook()
	log.Hooks.Add(hook)

	log.WithFields(logrus.Fields{
		"error": &customErr{expectedMsg},
	}).Error(unintendedMsg)

	select {
	case received := <-noticeChan:
		receivedErr := received.Errors[0]
		if receivedErr.Message != expectedMsg {
			t.Errorf("Unexpected message received: %s", receivedErr.Message)
		}
		if receivedErr.Type != expectedClass {
			t.Errorf("Unexpected error class: %s", receivedErr.Type)
		}
	case <-time.After(time.Second):
		t.Error("Timed out; no notice received by Airbrake API")
	}
}

// TestLogEntryWithNonErrorTypeNotReceived confirms that, when passing a
// non-error type using logrus.Fields, a HTTP server emulating an Airbrake
// endpoint receives the logrus.Entry.Message string.
//
// Only error types are supported when setting the 'error' field using
// logrus.WithFields().
func TestLogEntryWithNonErrorTypeNotReceived(t *testing.T) {
	log := logrus.New()
	hook := newTestHook()
	log.Hooks.Add(hook)

	log.WithFields(logrus.Fields{
		"error": expectedMsg,
	}).Error(unintendedMsg)

	select {
	case received := <-noticeChan:
		receivedErr := received.Errors[0]
		if receivedErr.Message != unintendedMsg {
			t.Errorf("Unexpected message received: %s", receivedErr.Message)
		}
	case <-time.After(time.Second):
		t.Error("Timed out; no notice received by Airbrake API")
	}
}

func TestLogEntryWithCustomFields(t *testing.T) {
	log := logrus.New()
	hook := newTestHook()
	log.Hooks.Add(hook)

	log.WithFields(logrus.Fields{
		"user_id": "123",
	}).Error(unintendedMsg)

	select {
	case received := <-noticeChan:
		receivedErr := received.Errors[0]
		if receivedErr.Message != unintendedMsg {
			t.Errorf("Unexpected message received: %s", receivedErr.Message)
		}
		if received.Context["user_id"] != "123" {
			t.Errorf("Expected message to contain Context[\"user_id\"] == \"123\" got %q", received.Context["user_id"])
		}
	case <-time.After(time.Second):
		t.Error("Timed out; no notice received by Airbrake API")
	}
}

// Returns a new airbrakeHook with the test server proxied
func newTestHook() *airbrakeHook {
	// Make a http.Client with the transport
	httpClient := &http.Client{Transport: &FakeRoundTripper{}}

	hook := NewHook(123, testAPIKey, "production")
	hook.Airbrake.Client = httpClient
	return hook
}

// gobrake API doesn't allow to override endpoint, we need a http.Roundtripper
type FakeRoundTripper struct {
}

func (rt *FakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	notice := &gobrake.Notice{}
	err = json.Unmarshal(b, notice)
	if err != nil {
		panic(err)
	}

	noticeChan <- notice

	jsonResponse := struct {
		Id string `json:"id"`
	}{"1"}

	sendResponse, _ := json.Marshal(jsonResponse)
	res := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       ioutil.NopCloser(bytes.NewReader(sendResponse)),
		Header:     make(http.Header),
	}
	return res, nil
}
