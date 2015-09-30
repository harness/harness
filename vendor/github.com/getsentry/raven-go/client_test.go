package raven

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

type testInterface struct{}

func (t *testInterface) Class() string   { return "sentry.interfaces.Test" }
func (t *testInterface) Culprit() string { return "codez" }

func TestPacketJSON(t *testing.T) {
	packet := &Packet{
		Project:    "1",
		EventID:    "2",
		Platform:   "linux",
		Culprit:    "caused_by",
		ServerName: "host1",
		Release:    "721e41770371db95eee98ca2707686226b993eda",
		Message:    "test",
		Timestamp:  Timestamp(time.Date(2000, 01, 01, 0, 0, 0, 0, time.UTC)),
		Level:      ERROR,
		Logger:     "com.getsentry.raven-go.logger-test-packet-json",
		Tags:       []Tag{Tag{"foo", "bar"}},
		Interfaces: []Interface{&Message{Message: "foo"}},
	}

	packet.AddTags(map[string]string{"foo": "foo"})
	packet.AddTags(map[string]string{"baz": "buzz"})

	expected := `{"message":"test","event_id":"2","project":"1","timestamp":"2000-01-01T00:00:00","level":"error","logger":"com.getsentry.raven-go.logger-test-packet-json","platform":"linux","culprit":"caused_by","server_name":"host1","release":"721e41770371db95eee98ca2707686226b993eda","tags":[["foo","bar"],["foo","foo"],["baz","buzz"]],"logentry":{"message":"foo"}}`
	actual := string(packet.JSON())

	if actual != expected {
		t.Errorf("incorrect json; got %s, want %s", actual, expected)
	}
}

func TestPacketInit(t *testing.T) {
	packet := &Packet{Message: "a", Interfaces: []Interface{&testInterface{}}}
	packet.Init("foo")

	if packet.Project != "foo" {
		t.Error("incorrect Project:", packet.Project)
	}
	if packet.Culprit != "codez" {
		t.Error("incorrect Culprit:", packet.Culprit)
	}
	if packet.ServerName == "" {
		t.Errorf("ServerName should not be empty")
	}
	if packet.Level != ERROR {
		t.Errorf("incorrect Level: got %d, want %d", packet.Level, ERROR)
	}
	if packet.Logger != "root" {
		t.Errorf("incorrect Logger: got %s, want %s", packet.Logger, "root")
	}
	if time.Time(packet.Timestamp).IsZero() {
		t.Error("Timestamp is zero")
	}
	if len(packet.EventID) != 32 {
		t.Error("incorrect EventID:", packet.EventID)
	}
}

func TestSetDSN(t *testing.T) {
	client := &Client{}
	client.SetDSN("https://u:p@example.com/sentry/1")

	if client.url != "https://example.com/sentry/api/1/store/" {
		t.Error("incorrect url:", client.url)
	}
	if client.projectID != "1" {
		t.Error("incorrect projectID:", client.projectID)
	}
	if client.authHeader != "Sentry sentry_version=4, sentry_key=u, sentry_secret=p" {
		t.Error("incorrect authHeader:", client.authHeader)
	}
}

func TestUnmarshalTag(t *testing.T) {
	actual := new(Tag)
	if err := json.Unmarshal([]byte(`["foo","bar"]`), actual); err != nil {
		t.Fatal("unable to decode JSON:", err)
	}

	expected := &Tag{Key: "foo", Value: "bar"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("incorrect Tag: wanted '%+v' and got '%+v'", expected, actual)
	}
}

func TestUnmarshalTags(t *testing.T) {
	tests := []struct {
		Input    string
		Expected Tags
	}{
		{
			`{"foo":"bar"}`,
			Tags{Tag{Key: "foo", Value: "bar"}},
		},
		{
			`[["foo","bar"],["bar","baz"]]`,
			Tags{Tag{Key: "foo", Value: "bar"}, Tag{Key: "bar", Value: "baz"}},
		},
	}

	for _, test := range tests {
		var actual Tags
		if err := json.Unmarshal([]byte(test.Input), &actual); err != nil {
			t.Fatal("unable to decode JSON:", err)
		}

		if !reflect.DeepEqual(actual, test.Expected) {
			t.Errorf("incorrect Tags: wanted '%+v' and got '%+v'", test.Expected, actual)
		}
	}
}

func TestMarshalTimestamp(t *testing.T) {
	timestamp := Timestamp(time.Date(2000, 01, 02, 03, 04, 05, 0, time.UTC))
	expected := `"2000-01-02T03:04:05"`

	actual, err := json.Marshal(timestamp)
	if err != nil {
		t.Error(err)
	}

	if string(actual) != expected {
		t.Errorf("incorrect string; got %s, want %s", actual, expected)
	}
}

func TestUnmarshalTimestamp(t *testing.T) {
	timestamp := `"2000-01-02T03:04:05"`
	expected := Timestamp(time.Date(2000, 01, 02, 03, 04, 05, 0, time.UTC))

	var actual Timestamp
	err := json.Unmarshal([]byte(timestamp), &actual)
	if err != nil {
		t.Error(err)
	}

	if actual != expected {
		t.Errorf("incorrect string; got %s, want %s", actual, expected)
	}
}
