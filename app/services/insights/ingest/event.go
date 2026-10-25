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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FlexString decodes a JSON string, number, bool, null, object or array into
// a string (composite values keep their JSON text). SDKs are not consistent
// about the types they send.
type FlexString string

// UnmarshalJSON implements json.Unmarshaler.
func (f *FlexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	switch {
	case len(b) == 0 || string(b) == "null":
		*f = ""
	case b[0] == '"':
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = FlexString(s)
	default:
		*f = FlexString(b)
	}
	return nil
}

// FlexInt decodes a JSON number or numeric string into an int. Anything else is 0.
type FlexInt int

// UnmarshalJSON implements json.Unmarshaler.
func (f *FlexInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		*f = FlexInt(v)
	} else {
		*f = 0
	}
	return nil
}

// Timestamp accepts Unix seconds (float, number or numeric string; values
// above 1e11 are taken as milliseconds) or RFC3339 strings.
type Timestamp struct{ time.Time }

var timestampLayouts = []string{
	time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		if v > 1e11 {
			v /= 1000
		}
		sec := int64(v)
		t.Time = time.Unix(sec, int64((v-float64(sec))*1e9)).UTC()
		return nil
	}
	for _, l := range timestampLayouts {
		if p, err := time.Parse(l, s); err == nil {
			t.Time = p.UTC()
			return nil
		}
	}
	return fmt.Errorf("invalid timestamp %q", s)
}

// MarshalJSON implements json.Marshaler.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.UTC().Format(time.RFC3339Nano))
}

// StringMap decodes either a JSON object or an array of [key, value] pairs
// (the two forms of tags and headers) into a map. Values are stringified.
type StringMap map[string]string

// UnmarshalJSON implements json.Unmarshaler.
func (m *StringMap) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	out := StringMap{}
	switch {
	case len(b) == 0 || string(b) == "null":
		*m = nil
		return nil
	case b[0] == '{':
		var o map[string]FlexString
		if err := json.Unmarshal(b, &o); err != nil {
			return err
		}
		for k, v := range o {
			out[k] = string(v)
		}
	case b[0] == '[':
		var arr []json.RawMessage
		if err := json.Unmarshal(b, &arr); err != nil {
			return err
		}
		for _, raw := range arr {
			var pair []FlexString
			if err := json.Unmarshal(raw, &pair); err != nil || len(pair) < 1 {
				continue // lenient: skip malformed pairs
			}
			v := ""
			if len(pair) > 1 {
				v = string(pair[1])
			}
			out[string(pair[0])] = v
		}
	default:
		// lenient: unknown shapes are dropped instead of failing the event
		*m = nil
		return nil
	}
	*m = out
	return nil
}

// LogEntry is the message of an event: either a plain string or an object
// {message, formatted, params}.
type LogEntry struct {
	// Message is the template (e.g. "user %s not found").
	Message string `json:"message,omitempty"`
	// Formatted is the rendered message.
	Formatted string `json:"formatted,omitempty"`
	Params    any    `json:"params,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (l *LogEntry) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		l.Message = s
		return nil
	}
	var aux struct {
		Message   FlexString `json:"message"`
		Formatted FlexString `json:"formatted"`
		Params    any        `json:"params"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	l.Message, l.Formatted, l.Params = string(aux.Message), string(aux.Formatted), aux.Params
	return nil
}

// Frame is a stack frame. Frames are ordered oldest call first (the last
// frame is where the exception was raised), as sent by SDKs.
type Frame struct {
	Filename    string         `json:"filename,omitempty"`
	AbsPath     string         `json:"abs_path,omitempty"`
	Module      string         `json:"module,omitempty"`
	Package     string         `json:"package,omitempty"`
	Function    string         `json:"function,omitempty"`
	Lineno      FlexInt        `json:"lineno,omitempty"`
	Colno       FlexInt        `json:"colno,omitempty"`
	InApp       *bool          `json:"in_app,omitempty"`
	ContextLine string         `json:"context_line,omitempty"`
	PreContext  []string       `json:"pre_context,omitempty"`
	PostContext []string       `json:"post_context,omitempty"`
	Vars        map[string]any `json:"vars,omitempty"`
}

// UnmarshalJSON tolerates non-string scalar metadata.
func (f *Frame) UnmarshalJSON(b []byte) error {
	type alias Frame
	aux := struct {
		Filename FlexString `json:"filename"`
		AbsPath  FlexString `json:"abs_path"`
		Module   FlexString `json:"module"`
		Package  FlexString `json:"package"`
		Function FlexString `json:"function"`
		*alias
	}{alias: (*alias)(f)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	f.Filename, f.AbsPath, f.Module = string(aux.Filename), string(aux.AbsPath), string(aux.Module)
	f.Package, f.Function = string(aux.Package), string(aux.Function)
	return nil
}

// Stacktrace is a list of frames.
type Stacktrace struct {
	Frames []Frame `json:"frames,omitempty"`
}

// Mechanism describes how an exception was captured.
type Mechanism struct {
	Type        string         `json:"type,omitempty"`
	Handled     *bool          `json:"handled,omitempty"`
	Description string         `json:"description,omitempty"`
	Synthetic   bool           `json:"synthetic,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}

// Exception is one entry of exception.values. Values are ordered oldest
// first: the last one is the outermost exception (the one that was thrown).
type Exception struct {
	Type       string      `json:"type,omitempty"`
	Value      string      `json:"value,omitempty"`
	Module     string      `json:"module,omitempty"`
	ThreadID   string      `json:"thread_id,omitempty"`
	Mechanism  *Mechanism  `json:"mechanism,omitempty"`
	Stacktrace *Stacktrace `json:"stacktrace,omitempty"`
}

// UnmarshalJSON tolerates non-string scalar metadata.
func (e *Exception) UnmarshalJSON(b []byte) error {
	type alias Exception
	aux := struct {
		Type     FlexString `json:"type"`
		Value    FlexString `json:"value"`
		Module   FlexString `json:"module"`
		ThreadID FlexString `json:"thread_id"`
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	e.Type, e.Value, e.Module, e.ThreadID = string(aux.Type), string(aux.Value), string(aux.Module), string(aux.ThreadID)
	return nil
}

// ExceptionList is exception: {"values": [...]}; a bare array or a single
// object are accepted too.
type ExceptionList struct {
	Values []Exception `json:"values,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (l *ExceptionList) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	switch b[0] {
	case '[':
		return json.Unmarshal(b, &l.Values)
	case '{':
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(b, &probe); err != nil {
			return err
		}
		if raw, ok := probe["values"]; ok {
			return json.Unmarshal(raw, &l.Values)
		}
		var one Exception
		if err := json.Unmarshal(b, &one); err != nil {
			return err
		}
		l.Values = []Exception{one}
		return nil
	}
	return fmt.Errorf("invalid exception")
}

// User is the affected user.
type User struct {
	ID        string         `json:"id,omitempty"`
	Email     string         `json:"email,omitempty"`
	Username  string         `json:"username,omitempty"`
	Name      string         `json:"name,omitempty"`
	IPAddress string         `json:"ip_address,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// UnmarshalJSON tolerates non-string scalars (numeric ids).
func (u *User) UnmarshalJSON(b []byte) error {
	type alias User
	aux := struct {
		ID        FlexString `json:"id"`
		Email     FlexString `json:"email"`
		Username  FlexString `json:"username"`
		Name      FlexString `json:"name"`
		IPAddress FlexString `json:"ip_address"`
		*alias
	}{alias: (*alias)(u)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	u.ID, u.Email, u.Username, u.Name, u.IPAddress =
		string(aux.ID), string(aux.Email), string(aux.Username), string(aux.Name), string(aux.IPAddress)
	return nil
}

// Request is the HTTP request attached to an event. QueryString, Cookies
// and Data may be strings or structured values depending on the SDK.
type Request struct {
	URL         string         `json:"url,omitempty"`
	Method      string         `json:"method,omitempty"`
	QueryString any            `json:"query_string,omitempty"`
	Cookies     any            `json:"cookies,omitempty"`
	Headers     StringMap      `json:"headers,omitempty"`
	Data        any            `json:"data,omitempty"`
	Env         map[string]any `json:"env,omitempty"`
}

// SDK identifies the client SDK.
type SDK struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// Event is an error event as sent by Sentry SDKs.
type Event struct {
	EventID     string          `json:"event_id,omitempty"`
	Timestamp   Timestamp       `json:"timestamp,omitzero"`
	Platform    string          `json:"platform,omitempty"`
	Level       string          `json:"level,omitempty"`
	Logger      string          `json:"logger,omitempty"`
	Release     string          `json:"release,omitempty"`
	Dist        string          `json:"dist,omitempty"`
	Environment string          `json:"environment,omitempty"`
	ServerName  string          `json:"server_name,omitempty"`
	Transaction string          `json:"transaction,omitempty"`
	Message     *LogEntry       `json:"message,omitempty"`
	LogEntry    *LogEntry       `json:"logentry,omitempty"`
	Exception   *ExceptionList  `json:"exception,omitempty"`
	Tags        StringMap       `json:"tags,omitempty"`
	User        *User           `json:"user,omitempty"`
	Request     *Request        `json:"request,omitempty"`
	Contexts    map[string]any  `json:"contexts,omitempty"`
	Extra       map[string]any  `json:"extra,omitempty"`
	Modules     StringMap       `json:"modules,omitempty"`
	SDK         *SDK            `json:"sdk,omitempty"`
	Breadcrumbs json.RawMessage `json:"breadcrumbs,omitempty"`
	// Fingerprint is the custom grouping rule set by the client.
	Fingerprint []string `json:"fingerprint,omitempty"`
}

// UnmarshalJSON decodes an event leniently: scalar metadata of any JSON type
// is stringified and unknown fields are ignored.
func (e *Event) UnmarshalJSON(b []byte) error {
	type alias Event
	aux := struct {
		EventID     FlexString   `json:"event_id"`
		Platform    FlexString   `json:"platform"`
		Level       FlexString   `json:"level"`
		Logger      FlexString   `json:"logger"`
		Release     FlexString   `json:"release"`
		Dist        FlexString   `json:"dist"`
		Environment FlexString   `json:"environment"`
		ServerName  FlexString   `json:"server_name"`
		Transaction FlexString   `json:"transaction"`
		Fingerprint []FlexString `json:"fingerprint"`
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	e.EventID, e.Platform, e.Level, e.Logger = string(aux.EventID), string(aux.Platform), string(aux.Level), string(aux.Logger)
	e.Release, e.Dist, e.Environment = string(aux.Release), string(aux.Dist), string(aux.Environment)
	e.ServerName, e.Transaction = string(aux.ServerName), string(aux.Transaction)
	e.Fingerprint = nil
	for _, f := range aux.Fingerprint {
		e.Fingerprint = append(e.Fingerprint, string(f))
	}
	return nil
}

// ParseEvent decodes a single event JSON document.
func ParseEvent(b []byte) (*Event, error) {
	var ev Event
	b = bytes.TrimSpace(b)
	if len(b) == 0 || b[0] != '{' {
		return nil, fmt.Errorf("%w: not a JSON object", ErrInvalidEvent)
	}
	if err := json.Unmarshal(b, &ev); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidEvent, err)
	}
	return &ev, nil
}

// MessageTemplate returns the unformatted message (logentry.message, then message).
func (e *Event) MessageTemplate() string {
	for _, l := range []*LogEntry{e.LogEntry, e.Message} {
		if l != nil && l.Message != "" {
			return l.Message
		}
	}
	return ""
}

// FormattedMessage returns the rendered message, falling back to the template.
func (e *Event) FormattedMessage() string {
	for _, l := range []*LogEntry{e.LogEntry, e.Message} {
		if l != nil && l.Formatted != "" {
			return l.Formatted
		}
	}
	return e.MessageTemplate()
}

// ExceptionValues returns exception.values (possibly empty).
func (e *Event) ExceptionValues() []Exception {
	if e.Exception == nil {
		return nil
	}
	return e.Exception.Values
}
