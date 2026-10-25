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

package ingest

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func parseFixture(t *testing.T, name string) *Envelope {
	t.Helper()
	env, err := ParseEnvelope(bytes.NewReader(fixture(t, name)), "", Limits{})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return env
}

func TestDSN(t *testing.T) {
	tests := []struct {
		in      string
		want    DSN
		wantErr bool
	}{
		{in: "https://abc@insights.example.com/42", want: DSN{Scheme: "https", PublicKey: "abc", Host: "insights.example.com", ProjectID: "42"}},
		{in: "http://abc:sec@localhost:3000/7", want: DSN{Scheme: "http", PublicKey: "abc", SecretKey: "sec", Host: "localhost:3000", ProjectID: "7"}},
		{in: "https://abc@host/prefix/sub/9", want: DSN{Scheme: "https", PublicKey: "abc", Host: "host", PathPrefix: "prefix/sub", ProjectID: "9"}},
		{in: "https://host/1", wantErr: true},
		{in: "https://abc@host", wantErr: true},
		{in: "ftp://abc@host/1", wantErr: true},
		{in: "garbage", wantErr: true},
	}
	for _, tt := range tests {
		got, err := ParseDSN(tt.in)
		if (err != nil) != tt.wantErr {
			t.Fatalf("%q: err=%v", tt.in, err)
		}
		if tt.wantErr {
			if !errors.Is(err, ErrInvalidDSN) || strings.Contains(err.Error(), "sec@") {
				t.Errorf("%q: bad error %v", tt.in, err)
			}
			continue
		}
		if got != tt.want {
			t.Errorf("%q: got %+v want %+v", tt.in, got, tt.want)
		}
	}
	if s := FormatDSN("https", "k", "h.io", "5"); s != "https://k@h.io/5" {
		t.Errorf("format: %s", s)
	}
	d, _ := ParseDSN("https://k@h.io/p/5")
	if d.String() != "https://k@h.io/p/5" {
		t.Errorf("roundtrip: %s", d)
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !ConstantTimeEqual("abc", "abc") || ConstantTimeEqual("abc", "abd") || ConstantTimeEqual("abc", "abcd") || ConstantTimeEqual("", "a") {
		t.Fatal("wrong result")
	}
}

func TestAuthenticate(t *testing.T) {
	const dsn = "https://envkey@host/42"
	tests := []struct {
		name    string
		in      AuthInput
		want    Credentials
		wantErr error
	}{
		{"header", AuthInput{ProjectID: "42", SentryAuth: "Sentry sentry_key=hdrkey, sentry_version=7, sentry_client=sentry.go/0.31"},
			Credentials{"42", "hdrkey", "header"}, nil},
		{"header wins over query", AuthInput{ProjectID: "42", SentryAuth: "Sentry sentry_key=hdrkey", Query: url.Values{"sentry_key": {"q"}}},
			Credentials{"42", "hdrkey", "header"}, nil},
		{"query", AuthInput{ProjectID: "42", Query: url.Values{"sentry_key": {"qkey"}, "sentry_version": {"7"}}},
			Credentials{"42", "qkey", "query"}, nil},
		{"envelope dsn", AuthInput{ProjectID: "42", EnvelopeDSN: dsn}, Credentials{"42", "envkey", "envelope"}, nil},
		{"envelope dsn without path project", AuthInput{EnvelopeDSN: dsn}, Credentials{"42", "envkey", "envelope"}, nil},
		{"project mismatch", AuthInput{ProjectID: "43", EnvelopeDSN: dsn}, Credentials{}, ErrProjectMismatch},
		{"none", AuthInput{ProjectID: "42"}, Credentials{}, ErrMissingKey},
		{"header without key", AuthInput{ProjectID: "42", SentryAuth: "Sentry sentry_version=7"}, Credentials{}, ErrMissingKey},
		{"bad dsn only", AuthInput{ProjectID: "42", EnvelopeDSN: "nope"}, Credentials{}, ErrInvalidDSN},
		{"bad dsn ignored when header present", AuthInput{ProjectID: "42", SentryAuth: "Sentry sentry_key=k", EnvelopeDSN: "nope"},
			Credentials{"42", "k", "header"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Authenticate(tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err=%v want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestParseEnvelopeFixtures(t *testing.T) {
	tests := []struct {
		file        string
		wantEvents  int
		wantItems   int
		wantSkipped map[string]int
		check       func(t *testing.T, env *Envelope)
	}{
		{"sentry_go_error.envelope", 1, 1, map[string]int{}, func(t *testing.T, env *Envelope) {
			ev := env.Events[0]
			if env.Header.DSN != "https://abc123publickey@insights.example.com/42" {
				t.Errorf("dsn: %s", env.Header.DSN)
			}
			if ev.Platform != "go" || ev.Release != "myapp@1.4.2" || ev.Environment != "production" || ev.ServerName != "api-7f9c" {
				t.Errorf("metadata: %+v", ev)
			}
			if !ev.Timestamp.Equal(time.Date(2026, 3, 1, 10, 15, 30, 123456000, time.UTC)) {
				t.Errorf("timestamp: %v", ev.Timestamp)
			}
			fr := ev.ExceptionValues()[0].Stacktrace.Frames
			if len(fr) != 4 || fr[2].Lineno != 88 || fr[2].Vars["o"] == nil || len(fr[2].PreContext) != 1 || fr[2].ContextLine == "" {
				t.Errorf("frames: %+v", fr)
			}
			if ev.Tags["region"] != "eu-west-1" || ev.User.ID != "u-1001" || ev.Request.Method != "POST" {
				t.Errorf("tags/user/request: %+v %+v %+v", ev.Tags, ev.User, ev.Request)
			}
		}},
		{"sentry_javascript_error.envelope", 1, 1, map[string]int{}, func(t *testing.T, env *Envelope) {
			ev := env.Events[0]
			if ev.Timestamp.Unix() != 1772360130 || ev.Timestamp.Nanosecond() != 500000000 {
				t.Errorf("float timestamp: %v", ev.Timestamp)
			}
			if ev.Tags["browser"] != "chrome" || ev.Tags["feature"] != "checkout" {
				t.Errorf("array tags: %v", ev.Tags)
			}
			m := ev.ExceptionValues()[0].Mechanism
			if m.Type != "onerror" || m.Handled == nil || *m.Handled {
				t.Errorf("mechanism: %+v", m)
			}
			if ev.Request.Headers["User-Agent"] == "" {
				t.Errorf("headers: %v", ev.Request.Headers)
			}
		}},
		{"sentry_javascript_transaction.envelope", 0, 1, map[string]int{"transaction": 1}, nil},
		{"sentry_python_error.envelope", 1, 3, map[string]int{"session": 1, "client_report": 1}, func(t *testing.T, env *Envelope) {
			ev := env.Events[0]
			if ev.User.ID != "77" || ev.Tags["attempt"] != "3" {
				t.Errorf("flex types: user=%+v tags=%v", ev.User, ev.Tags)
			}
			if len(ev.ExceptionValues()) != 2 || ev.ExceptionValues()[1].Type != "RuntimeError" || ev.Logger != "app.tasks" {
				t.Errorf("exceptions: %+v", ev.ExceptionValues())
			}
		}},
		{"attachment.envelope", 1, 2, map[string]int{"attachment": 1}, func(t *testing.T, env *Envelope) {
			a := env.Items[1]
			if a.Filename != "core.bin" || len(a.Payload) != 40+9 || !bytes.Contains(a.Payload, []byte("\n")) {
				t.Errorf("attachment: %q len=%d", a.Filename, len(a.Payload))
			}
			if env.Events[0].Level != "fatal" || env.Events[0].Message.Message != "boom" {
				t.Errorf("event: %+v", env.Events[0])
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			env := parseFixture(t, tt.file)
			if len(env.Events) != tt.wantEvents || len(env.Items) != tt.wantItems {
				t.Fatalf("events=%d items=%d", len(env.Events), len(env.Items))
			}
			if !reflect.DeepEqual(env.Skipped, tt.wantSkipped) {
				t.Errorf("skipped=%v want %v", env.Skipped, tt.wantSkipped)
			}
			if len(env.ItemErrors) != 0 {
				t.Errorf("item errors: %v", env.ItemErrors)
			}
			if tt.check != nil {
				tt.check(t, env)
			}
		})
	}
}

func compress(t *testing.T, kind string, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	var w interface {
		Write([]byte) (int, error)
		Close() error
	}
	switch kind {
	case "gzip":
		w = gzip.NewWriter(&buf)
	case "deflate":
		w = zlib.NewWriter(&buf)
	case "rawdeflate":
		w, _ = flate.NewWriter(&buf, flate.DefaultCompression)
	}
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Bytes()
}

func TestParseEnvelopeEncodingsAndLimits(t *testing.T) {
	plain := fixture(t, "sentry_python_error.envelope")
	tests := []struct {
		name     string
		body     []byte
		encoding string
		lim      Limits
		wantErr  error
	}{
		{"identity", plain, "", Limits{}, nil},
		{"gzip", compress(t, "gzip", plain), "gzip", Limits{}, nil},
		{"gzip sniffed without header", compress(t, "gzip", plain), "", Limits{}, nil},
		{"deflate zlib", compress(t, "deflate", plain), "deflate", Limits{}, nil},
		{"deflate raw", compress(t, "rawdeflate", plain), "deflate", Limits{}, nil},
		{"unsupported", plain, "br", Limits{}, ErrInvalidEncoding},
		{"corrupt gzip", []byte("not gzip"), "gzip", Limits{}, ErrInvalidEncoding},
		{"compressed too large", compress(t, "gzip", plain), "gzip", Limits{MaxCompressedSize: 10}, ErrTooLarge},
		{"plain too large", plain, "", Limits{MaxCompressedSize: 10}, ErrTooLarge},
		{"decompressed too large (bomb)", compress(t, "gzip", append(plain, bytes.Repeat([]byte("\n"), 1<<20)...)), "gzip", Limits{MaxDecompressedSize: 4096}, ErrTooLarge},
		{"too many items", plain, "", Limits{MaxItems: 2}, ErrTooManyItems},
		{"empty", nil, "", Limits{}, ErrInvalidEnvelope},
		{"bad header", []byte("not json\n"), "", Limits{}, ErrInvalidEnvelope},
		{"bad item header", []byte("{}\nnot json\n{}\n"), "", Limits{}, ErrInvalidEnvelope},
		{"length exceeds body", []byte("{}\n{\"type\":\"event\",\"length\":99}\n{}\n"), "", Limits{}, ErrInvalidEnvelope},
		{"length without newline", []byte("{}\n{\"type\":\"event\",\"length\":2}\n{}xx\n"), "", Limits{}, ErrInvalidEnvelope},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := ParseEnvelope(bytes.NewReader(tt.body), tt.encoding, tt.lim)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err=%v want %v", err, tt.wantErr)
			}
			if err == nil && (len(env.Events) != 1 || env.Events[0].Platform != "python") {
				t.Errorf("bad parse: %+v", env)
			}
			if err != nil && !IsClientError(err) {
				t.Errorf("not a client error: %v", err)
			}
		})
	}
}

func TestParseEnvelopeEdgeCases(t *testing.T) {
	t.Run("no trailing newline, blank lines, event id from header", func(t *testing.T) {
		in := "{\"event_id\":\"abc\"}\n\n{\"type\":\"event\"}\n{\"message\":\"hi\"}"
		env, err := ParseEnvelopeBytes([]byte(in), Limits{})
		if err != nil || len(env.Events) != 1 || env.Events[0].EventID != "abc" || env.Events[0].Message.Message != "hi" {
			t.Fatalf("%v %+v", err, env)
		}
	})
	t.Run("header only", func(t *testing.T) {
		env, err := ParseEnvelopeBytes([]byte("{\"dsn\":\"https://k@h/1\"}\n"), Limits{})
		if err != nil || len(env.Items) != 0 || env.Header.DSN == "" {
			t.Fatalf("%v %+v", err, env)
		}
	})
	t.Run("invalid event does not abort envelope", func(t *testing.T) {
		in := "{}\n{\"type\":\"event\"}\n[1,2]\n{\"type\":\"event\"}\n{\"message\":\"ok\"}\n"
		env, err := ParseEnvelopeBytes([]byte(in), Limits{})
		if err != nil || len(env.Events) != 1 || len(env.ItemErrors) != 1 || !errors.Is(env.ItemErrors[0].Err, ErrInvalidEvent) {
			t.Fatalf("%v %+v", err, env)
		}
	})
	t.Run("unknown item types counted", func(t *testing.T) {
		in := "{}\n{\"type\":\"profile\"}\n{}\n{\"type\":\"profile\",\"length\":0}\n\n{\"type\":\"span\"}\n{}\n"
		env, err := ParseEnvelopeBytes([]byte(in), Limits{})
		if err != nil || env.Skipped["profile"] != 2 || env.Skipped["span"] != 1 || len(env.Events) != 0 {
			t.Fatalf("%v %+v", err, env)
		}
	})
}

func TestParseStore(t *testing.T) {
	body := fixture(t, "store_legacy.json")
	b64 := []byte(base64.StdEncoding.EncodeToString(compress(t, "deflate", body)))
	tests := []struct {
		name, enc string
		body      []byte
	}{
		{"plain", "", body},
		{"gzip", "gzip", compress(t, "gzip", body)},
		{"deflate", "deflate", compress(t, "deflate", body)},
		{"legacy base64 zlib", "", b64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, err := ParseStore(bytes.NewReader(tt.body), tt.enc, Limits{})
			if err != nil {
				t.Fatal(err)
			}
			if ev.EventID != "fc6d8c0c43fc4630ad850ee518f1b9d0" || ev.LogEntry.Message != "user %s not found" ||
				ev.FormattedMessage() != "user 1234 not found" || ev.Timestamp.Second() != 30 {
				t.Errorf("event: %+v", ev)
			}
		})
	}
	if _, err := ParseStore(strings.NewReader("???"), "", Limits{}); !errors.Is(err, ErrInvalidEvent) {
		t.Errorf("garbage: %v", err)
	}
	if _, err := ParseStore(strings.NewReader("[]"), "", Limits{}); !errors.Is(err, ErrInvalidEvent) {
		t.Errorf("array: %v", err)
	}
}

func TestEventLeniency(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		check func(t *testing.T, ev *Event)
	}{
		{"timestamp ms and string number", `{"timestamp":"1772360130"}`, func(t *testing.T, ev *Event) {
			if ev.Timestamp.Unix() != 1772360130 {
				t.Error(ev.Timestamp)
			}
		}},
		{"timestamp millis", `{"timestamp":1772360130500}`, func(t *testing.T, ev *Event) {
			if ev.Timestamp.Unix() != 1772360130 {
				t.Error(ev.Timestamp)
			}
		}},
		{"timestamp rfc3339 offset", `{"timestamp":"2026-03-01T11:15:30+01:00"}`, func(t *testing.T, ev *Event) {
			if ev.Timestamp.Hour() != 10 {
				t.Error(ev.Timestamp)
			}
		}},
		{"message object", `{"message":{"message":"a %s","formatted":"a b","params":["b"]}}`, func(t *testing.T, ev *Event) {
			if ev.MessageTemplate() != "a %s" || ev.FormattedMessage() != "a b" {
				t.Errorf("%+v", ev.Message)
			}
		}},
		{"exception as array", `{"exception":[{"type":"E","value":5}]}`, func(t *testing.T, ev *Event) {
			if len(ev.ExceptionValues()) != 1 || ev.ExceptionValues()[0].Value != "5" {
				t.Errorf("%+v", ev.Exception)
			}
		}},
		{"exception single object", `{"exception":{"type":"E"}}`, func(t *testing.T, ev *Event) {
			if len(ev.ExceptionValues()) != 1 || ev.ExceptionValues()[0].Type != "E" {
				t.Errorf("%+v", ev.Exception)
			}
		}},
		{"numeric metadata and fingerprint", `{"release":12,"environment":true,"fingerprint":["a",1]}`, func(t *testing.T, ev *Event) {
			if ev.Release != "12" || ev.Environment != "true" || !reflect.DeepEqual(ev.Fingerprint, []string{"a", "1"}) {
				t.Errorf("%+v", ev)
			}
		}},
		{"tags pairs with numbers and bad entries", `{"tags":[["a",1],["b"],"junk",["c","d"]]}`, func(t *testing.T, ev *Event) {
			if !reflect.DeepEqual(map[string]string(ev.Tags), map[string]string{"a": "1", "b": "", "c": "d"}) {
				t.Errorf("%v", ev.Tags)
			}
		}},
		{"tags object with non-string", `{"tags":{"n":1,"b":false}}`, func(t *testing.T, ev *Event) {
			if ev.Tags["n"] != "1" || ev.Tags["b"] != "false" {
				t.Errorf("%v", ev.Tags)
			}
		}},
		{"weird tags dropped", `{"tags":"nope","level":"info"}`, func(t *testing.T, ev *Event) {
			if ev.Tags != nil || ev.Level != "info" {
				t.Errorf("%+v", ev)
			}
		}},
		{"frame numeric strings", `{"exception":{"values":[{"type":"E","stacktrace":{"frames":[{"lineno":"12","colno":3.0,"function":"f","in_app":true}]}}]}}`, func(t *testing.T, ev *Event) {
			f := ev.ExceptionValues()[0].Stacktrace.Frames[0]
			if f.Lineno != 12 || f.Colno != 3 {
				t.Errorf("%+v", f)
			}
		}},
		{"unknown fields", `{"foo":{"bar":[1,2]},"level":"warning","debug_meta":{"images":[]}}`, func(t *testing.T, ev *Event) {
			if ev.Level != "warning" {
				t.Error(ev.Level)
			}
		}},
		{"headers as pairs, cookies map", `{"request":{"headers":[["Cookie","a=b"]],"cookies":{"a":"b"}}}`, func(t *testing.T, ev *Event) {
			if ev.Request.Headers["Cookie"] != "a=b" {
				t.Error(ev.Request.Headers)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, err := ParseEvent([]byte(tt.json))
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, ev)
		})
	}
	if _, err := ParseEvent([]byte(`{"timestamp":"yesterday"}`)); err == nil {
		t.Error("expected error for unparseable timestamp")
	}
}

func TestEventJSONRoundTrip(t *testing.T) {
	env := parseFixture(t, "sentry_go_error.envelope")
	b, err := json.Marshal(env.Events[0])
	if err != nil {
		t.Fatal(err)
	}
	ev2, err := ParseEvent(b)
	if err != nil {
		t.Fatal(err)
	}
	if ComputeFingerprint(env.Events[0]).Hash != ComputeFingerprint(ev2).Hash || !ev2.Timestamp.Equal(env.Events[0].Timestamp.Time) {
		t.Error("roundtrip changed event")
	}
}

func TestFingerprint(t *testing.T) {
	evFrom := func(t *testing.T, s string) *Event {
		ev, err := ParseEvent([]byte(s))
		if err != nil {
			t.Fatal(err)
		}
		return ev
	}
	tests := []struct {
		name       string
		json       string
		wantSource string
		wantComps  []string
	}{
		{"go in-app frames only, closure suffix stripped",
			string(fixtureEventJSON(t, "sentry_go_error.envelope")), SourceStacktrace,
			[]string{"type:*errors.errorString", "frame:main|main.main",
				"frame:github.com/acme/api/internal/orders|github.com/acme/api/internal/orders.(*Handler).Create",
				"frame:github.com/acme/api/internal/orders|github.com/acme/api/internal/orders.(*Handler).Create"}},
		{"js hashed bundle names normalized", string(fixtureEventJSON(t, "sentry_javascript_error.envelope")), SourceStacktrace,
			[]string{"type:TypeError", "frame:assets/index.js|Object.<anonymous>", "frame:assets/index.js|computeTotal"}},
		{"python outermost exception frames, module types", string(fixtureEventJSON(t, "sentry_python_error.envelope")), SourceStacktrace,
			[]string{"type:builtins.KeyError", "type:builtins.RuntimeError", "frame:app.tasks|send_invoice", "frame:app.tasks|<listcomp>"}},
		{"no in-app frames uses all",
			`{"exception":{"values":[{"type":"E","stacktrace":{"frames":[{"filename":"/usr/lib/a/b/c.py","function":"f","lineno":1},{"module":"m","function":"g"}]}}]}}`,
			SourceStacktrace, []string{"type:E", "frame:b/c.py|f", "frame:m|g"}},
		{"outermost exception without stacktrace falls back to inner",
			`{"exception":{"values":[{"type":"A","stacktrace":{"frames":[{"module":"m","function":"f","in_app":true}]}},{"type":"B","value":"x"}]}}`,
			SourceStacktrace, []string{"type:A", "type:B", "frame:m|f"}},
		{"exception without stacktrace uses normalized value",
			`{"exception":{"values":[{"type":"Error","value":"timeout after 30 seconds on 0xdeadbeef"}]}}`,
			SourceException, []string{"type:Error", "value:timeout after <num> seconds on <hex>"}},
		{"message template", `{"logentry":{"message":"user %s not found","formatted":"user 7 not found"}}`, SourceMessage, []string{"message:user %s not found"}},
		{"message normalization uuid hex numbers",
			`{"message":"order 123 for 550e8400-e29b-41d4-a716-446655440000 at 0xdeadbeef took 1.5s"}`,
			SourceMessage, []string{"message:order <num> for <uuid> at <hex> took <num>s"}},
		{"custom", `{"fingerprint":["a","b"],"message":"x"}`, SourceCustom, []string{"a", "b"}},
		{"custom with default", `{"fingerprint":["{{ default }}","tenant-1"],"message":"hello 5"}`, SourceCustom, []string{"message:hello <num>", "tenant-1"}},
		{"empty", `{}`, SourceEmpty, []string{"<no-data>"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := evFrom(t, tt.json)
			fp := ComputeFingerprint(ev)
			if fp.Source != tt.wantSource || !reflect.DeepEqual(fp.Components, tt.wantComps) {
				t.Errorf("got %s %q\nwant %s %q", fp.Source, fp.Components, tt.wantSource, tt.wantComps)
			}
			if len(fp.Hash) != 32 || fp.Hash != ComputeFingerprint(ev).Hash {
				t.Errorf("hash not stable: %s", fp.Hash)
			}
		})
	}
}

func fixtureEventJSON(t *testing.T, name string) []byte {
	t.Helper()
	env := parseFixture(t, name)
	for _, it := range env.Items {
		if it.Type == ItemTypeEvent {
			return it.Payload
		}
	}
	t.Fatal("no event")
	return nil
}

func TestFingerprintStability(t *testing.T) {
	base := `{"exception":{"values":[{"type":"E","stacktrace":{"frames":[{"filename":"/a/b/c.go","function":"pkg.F.func1","lineno":%d,"in_app":true}]}}]}}`
	hash := func(s string) string {
		ev, err := ParseEvent([]byte(s))
		if err != nil {
			t.Fatal(err)
		}
		return ComputeFingerprint(ev).Hash
	}
	same := []struct{ name, a, b string }{
		{"line numbers ignored", strings.Replace(base, "%d", "10", 1), strings.Replace(base, "%d", "99", 1)},
		{"deploy path ignored", base, strings.Replace(base, "/a/b/c.go", "/other/root/b/c.go", 1)},
		{"go closure numbering", base, strings.Replace(base, "func1", "func2.3", 1)},
		{"message numbers", `{"message":"retry 3 failed"}`, `{"message":"retry 4 failed"}`},
		{"java lambda", `{"exception":{"values":[{"type":"E","stacktrace":{"frames":[{"module":"a.B","function":"lambda$run$0","in_app":true}]}}]}}`,
			`{"exception":{"values":[{"type":"E","stacktrace":{"frames":[{"module":"a.B","function":"lambda$run$7","in_app":true}]}}]}}`},
		{"address in function", `{"exception":{"values":[{"type":"E","stacktrace":{"frames":[{"module":"m","function":"fn 0x7ffd1000","in_app":true}]}}]}}`,
			`{"exception":{"values":[{"type":"E","stacktrace":{"frames":[{"module":"m","function":"fn 0x7ffd2abc","in_app":true}]}}]}}`},
	}
	for _, tt := range same {
		if hash(strings.Replace(tt.a, "%d", "1", 1)) != hash(strings.Replace(tt.b, "%d", "1", 1)) {
			t.Errorf("%s: hashes differ", tt.name)
		}
	}
	diff := []struct{ name, a, b string }{
		{"different function", base, strings.Replace(base, "pkg.F", "pkg.G", 1)},
		{"different type", base, strings.Replace(base, `"E"`, `"X"`, 1)},
		{"different message", `{"message":"a"}`, `{"message":"b"}`},
	}
	for _, tt := range diff {
		if hash(strings.Replace(tt.a, "%d", "1", 1)) == hash(strings.Replace(tt.b, "%d", "1", 1)) {
			t.Errorf("%s: hashes equal", tt.name)
		}
	}
}

func TestSanitize(t *testing.T) {
	env := parseFixture(t, "sentry_go_error.envelope")
	ev := env.Events[0]
	before := ComputeFingerprint(ev).Hash
	Sanitize(ev, DefaultSanitizeOptions())
	for _, f := range ev.ExceptionValues()[0].Stacktrace.Frames {
		if f.Vars != nil {
			t.Errorf("vars kept: %v", f.Vars)
		}
	}
	h := ev.Request.Headers
	if h["Authorization"] != Filtered || h["Cookie"] != Filtered || h["Content-Type"] != "application/json" {
		t.Errorf("headers: %v", h)
	}
	if ev.Request.Cookies != Filtered {
		t.Errorf("cookies: %v", ev.Request.Cookies)
	}
	if ComputeFingerprint(ev).Hash != before {
		t.Error("sanitization changed the fingerprint")
	}

	t.Run("cookie map and zero options", func(t *testing.T) {
		e, _ := ParseEvent([]byte(`{"request":{"cookies":{"a":"1"},"headers":{"X":"y"}}}`))
		Sanitize(e, SanitizeOptions{})
		if e.Request.Cookies.(map[string]any)["a"] != "1" || e.Request.Headers["X"] != "y" {
			t.Error("zero options must not change anything")
		}
		Sanitize(e, SanitizeOptions{ScrubCookies: true})
		if e.Request.Cookies.(map[string]any)["a"] != Filtered || e.Request.Headers["X"] != "y" {
			t.Errorf("%+v", e.Request)
		}
	})
	t.Run("no request nor exception", func(t *testing.T) {
		Sanitize(&Event{}, DefaultSanitizeOptions())
	})
}
