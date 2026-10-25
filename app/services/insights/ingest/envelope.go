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
	"fmt"
	"io"
	"strings"
)

// Item types.
const (
	ItemTypeEvent = "event"
)

// Limits bound the work done for one request.
type Limits struct {
	// MaxCompressedSize is the maximum body size as received (default 1 MiB).
	MaxCompressedSize int64
	// MaxDecompressedSize is the maximum size after decompression (default 20 MiB).
	MaxDecompressedSize int64
	// MaxItems is the maximum number of envelope items (default 100).
	MaxItems int
}

// DefaultLimits returns the default limits.
func DefaultLimits() Limits {
	return Limits{MaxCompressedSize: 1 << 20, MaxDecompressedSize: 20 << 20, MaxItems: 100}
}

func (l Limits) withDefaults() Limits {
	d := DefaultLimits()
	if l.MaxCompressedSize <= 0 {
		l.MaxCompressedSize = d.MaxCompressedSize
	}
	if l.MaxDecompressedSize <= 0 {
		l.MaxDecompressedSize = d.MaxDecompressedSize
	}
	if l.MaxItems <= 0 {
		l.MaxItems = d.MaxItems
	}
	return l
}

// EnvelopeHeader is the first line of an envelope.
type EnvelopeHeader struct {
	EventID string
	// DSN is the "dsn" field, usable for authentication.
	DSN    string
	SentAt Timestamp
}

// Item is one envelope item. Payload aliases the decompressed body.
type Item struct {
	Type        string
	ContentType string
	Filename    string
	Payload     []byte
}

// ItemError records an item that could not be decoded; parsing continues.
type ItemError struct {
	Index int
	Type  string
	Err   error
}

// Envelope is a parsed envelope.
type Envelope struct {
	Header EnvelopeHeader
	// Items are all items in order, including skipped ones (raw payloads).
	Items []Item
	// Events are the decoded "event" items.
	Events []*Event
	// Counts is the number of items per type.
	Counts map[string]int
	// Skipped is the number of non-event items per type (transaction, session,
	// attachment, client_report, ...), which the caller may drop.
	Skipped    map[string]int
	ItemErrors []ItemError
}

// ReadBody reads a request body enforcing the limits and removing the
// Content-Encoding (gzip, deflate or identity). A gzip stream is also
// detected by magic number when the header is missing.
func ReadBody(r io.Reader, contentEncoding string, lim Limits) ([]byte, error) {
	lim = lim.withDefaults()
	raw, err := io.ReadAll(io.LimitReader(r, lim.MaxCompressedSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > lim.MaxCompressedSize {
		return nil, fmt.Errorf("%w: compressed body exceeds %d bytes", ErrTooLarge, lim.MaxCompressedSize)
	}
	enc := strings.ToLower(strings.TrimSpace(contentEncoding))
	if enc == "" || enc == "identity" {
		if len(raw) > 2 && raw[0] == 0x1f && raw[1] == 0x8b {
			enc = "gzip"
		} else {
			enc = "identity"
		}
	}
	var dr io.Reader
	switch enc {
	case "identity":
		if int64(len(raw)) > lim.MaxDecompressedSize {
			return nil, fmt.Errorf("%w: body exceeds %d bytes", ErrTooLarge, lim.MaxDecompressedSize)
		}
		return raw, nil
	case "gzip", "x-gzip":
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("%w: gzip: %v", ErrInvalidEncoding, err)
		}
		dr = zr
	case "deflate":
		// HTTP deflate is zlib-wrapped, but some clients send raw deflate.
		if zr, err := zlib.NewReader(bytes.NewReader(raw)); err == nil {
			dr = zr
		} else {
			dr = flate.NewReader(bytes.NewReader(raw))
		}
	default:
		return nil, fmt.Errorf("%w: unsupported %q", ErrInvalidEncoding, contentEncoding)
	}
	out, err := io.ReadAll(io.LimitReader(dr, lim.MaxDecompressedSize+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidEncoding, err)
	}
	if int64(len(out)) > lim.MaxDecompressedSize {
		return nil, fmt.Errorf("%w: decompressed body exceeds %d bytes", ErrTooLarge, lim.MaxDecompressedSize)
	}
	return out, nil
}

// ParseEnvelope reads, decompresses and parses a request body of
// POST /api/{project}/envelope/.
func ParseEnvelope(r io.Reader, contentEncoding string, lim Limits) (*Envelope, error) {
	body, err := ReadBody(r, contentEncoding, lim)
	if err != nil {
		return nil, err
	}
	return ParseEnvelopeBytes(body, lim)
}

// ParseEnvelopeBytes parses an already decompressed envelope.
func ParseEnvelopeBytes(data []byte, lim Limits) (*Envelope, error) {
	lim = lim.withDefaults()
	pos := 0
	line := func() []byte {
		i := bytes.IndexByte(data[pos:], '\n')
		var l []byte
		if i < 0 {
			l, pos = data[pos:], len(data)
		} else {
			l, pos = data[pos:pos+i], pos+i+1
		}
		return l
	}

	first := bytes.TrimSpace(line())
	if len(first) == 0 {
		return nil, fmt.Errorf("%w: missing header", ErrInvalidEnvelope)
	}
	var hdr struct {
		EventID FlexString      `json:"event_id"`
		DSN     FlexString      `json:"dsn"`
		SentAt  json.RawMessage `json:"sent_at"`
	}
	if err := json.Unmarshal(first, &hdr); err != nil {
		return nil, fmt.Errorf("%w: header: %v", ErrInvalidEnvelope, err)
	}
	env := &Envelope{
		Header: EnvelopeHeader{EventID: string(hdr.EventID), DSN: string(hdr.DSN)},
		Counts: map[string]int{}, Skipped: map[string]int{},
	}
	if len(hdr.SentAt) > 0 {
		_ = json.Unmarshal(hdr.SentAt, &env.Header.SentAt) // advisory only
	}

	for pos < len(data) {
		ih := bytes.TrimSpace(line())
		if len(ih) == 0 {
			continue // tolerate blank lines between items
		}
		if len(env.Items) >= lim.MaxItems {
			return nil, ErrTooManyItems
		}
		var h struct {
			Type        FlexString `json:"type"`
			Length      *float64   `json:"length"`
			ContentType FlexString `json:"content_type"`
			Filename    FlexString `json:"filename"`
		}
		if err := json.Unmarshal(ih, &h); err != nil {
			return nil, fmt.Errorf("%w: item header: %v", ErrInvalidEnvelope, err)
		}
		var payload []byte
		if h.Length != nil {
			n := int(*h.Length)
			if n < 0 || n > len(data)-pos {
				return nil, fmt.Errorf("%w: item length %d exceeds body", ErrInvalidEnvelope, n)
			}
			payload, pos = data[pos:pos+n], pos+n
			if pos < len(data) {
				if data[pos] != '\n' {
					return nil, fmt.Errorf("%w: item not followed by newline", ErrInvalidEnvelope)
				}
				pos++
			}
		} else {
			payload = line()
		}
		it := Item{Type: string(h.Type), ContentType: string(h.ContentType), Filename: string(h.Filename), Payload: payload}
		idx := len(env.Items)
		env.Items = append(env.Items, it)
		env.Counts[it.Type]++
		if it.Type != ItemTypeEvent {
			env.Skipped[it.Type]++
			continue
		}
		ev, err := ParseEvent(payload)
		if err != nil {
			env.ItemErrors = append(env.ItemErrors, ItemError{Index: idx, Type: it.Type, Err: err})
			continue
		}
		if ev.EventID == "" {
			ev.EventID = env.Header.EventID
		}
		env.Events = append(env.Events, ev)
	}
	return env, nil
}

// ParseStore parses the body of the legacy POST /api/{project}/store/: a JSON
// event, optionally compressed (also the old base64+zlib form).
func ParseStore(r io.Reader, contentEncoding string, lim Limits) (*Event, error) {
	body, err := ReadBody(r, contentEncoding, lim)
	if err != nil {
		return nil, err
	}
	t := bytes.TrimSpace(body)
	if len(t) > 0 && t[0] != '{' {
		// legacy raven clients: base64(zlib(json))
		dec, derr := base64.StdEncoding.DecodeString(string(t))
		if derr != nil {
			return nil, fmt.Errorf("%w: not JSON", ErrInvalidEvent)
		}
		zr, zerr := zlib.NewReader(bytes.NewReader(dec))
		if zerr != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidEncoding, zerr)
		}
		lm := lim.withDefaults()
		t, err = io.ReadAll(io.LimitReader(zr, lm.MaxDecompressedSize+1))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidEncoding, err)
		}
		if int64(len(t)) > lm.MaxDecompressedSize {
			return nil, ErrTooLarge
		}
	}
	return ParseEvent(t)
}

// IsClientError reports whether err stems from a bad request (as opposed to an internal failure).
func IsClientError(err error) bool {
	for _, e := range []error{ErrInvalidDSN, ErrMissingKey, ErrProjectMismatch, ErrTooLarge,
		ErrTooManyItems, ErrInvalidEnvelope, ErrInvalidEvent, ErrInvalidEncoding} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
