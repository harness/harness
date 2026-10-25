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
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net/url"
	"strings"
)

// DSN is a parsed Sentry DSN: {scheme}://{public_key}@{host}/{project_id}.
type DSN struct {
	Scheme    string
	PublicKey string
	// SecretKey is the legacy secret part ({public}:{secret}@host); optional.
	SecretKey string
	Host      string // includes port
	// PathPrefix is an optional path before the project id (no leading/trailing slash).
	PathPrefix string
	ProjectID  string
}

// ParseDSN parses a DSN string.
func ParseDSN(raw string) (DSN, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User == nil {
		return DSN{}, fmt.Errorf("%w: %q", ErrInvalidDSN, redactDSN(raw))
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return DSN{}, fmt.Errorf("%w: unsupported scheme %q", ErrInvalidDSN, u.Scheme)
	}
	key := u.User.Username()
	if key == "" {
		return DSN{}, fmt.Errorf("%w: empty public key", ErrInvalidDSN)
	}
	secret, _ := u.User.Password()
	p := strings.Trim(u.Path, "/")
	if p == "" {
		return DSN{}, fmt.Errorf("%w: missing project id", ErrInvalidDSN)
	}
	prefix, project := "", p
	if i := strings.LastIndex(p, "/"); i >= 0 {
		prefix, project = p[:i], p[i+1:]
	}
	return DSN{
		Scheme: u.Scheme, PublicKey: key, SecretKey: secret,
		Host: u.Host, PathPrefix: prefix, ProjectID: project,
	}, nil
}

// redactDSN hides the credentials so errors never echo keys.
func redactDSN(raw string) string {
	if i := strings.Index(raw, "@"); i >= 0 {
		if j := strings.Index(raw, "://"); j >= 0 && j < i {
			return raw[:j+3] + "***" + raw[i:]
		}
	}
	return raw
}

// String formats the DSN without the secret key.
func (d DSN) String() string {
	var b strings.Builder
	b.WriteString(d.Scheme)
	b.WriteString("://")
	b.WriteString(d.PublicKey)
	b.WriteString("@")
	b.WriteString(d.Host)
	b.WriteString("/")
	if d.PathPrefix != "" {
		b.WriteString(d.PathPrefix)
		b.WriteString("/")
	}
	b.WriteString(d.ProjectID)
	return b.String()
}

// FormatDSN builds the DSN handed to SDKs.
func FormatDSN(scheme, publicKey, host, projectID string) string {
	return DSN{Scheme: scheme, PublicKey: publicKey, Host: host, ProjectID: projectID}.String()
}

// ConstantTimeEqual compares two secrets without leaking their length or content through timing.
func ConstantTimeEqual(a, b string) bool {
	ha, hb := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}

// Credentials identify the project and the key presented by a client.
// The caller must validate PublicKey against the stored project key
// (see ConstantTimeEqual).
type Credentials struct {
	ProjectID string
	PublicKey string
	// Source is one of "header", "query" or "envelope".
	Source string
}

// AuthInput carries everything a request may use to authenticate.
type AuthInput struct {
	// ProjectID is the project id from the request path.
	ProjectID string
	// SentryAuth is the X-Sentry-Auth header (or an "Authorization: Sentry ..." value).
	SentryAuth string
	// Query holds the URL query parameters (sentry_key).
	Query url.Values
	// EnvelopeDSN is the "dsn" field of the envelope header, if parsed already.
	EnvelopeDSN string
}

// ParseSentryAuth extracts the key/value pairs of a
// "Sentry sentry_key=..., sentry_version=7, sentry_client=..." header.
func ParseSentryAuth(h string) map[string]string {
	h = strings.TrimSpace(h)
	if len(h) >= 7 && strings.EqualFold(h[:7], "sentry ") {
		h = h[7:]
	}
	out := map[string]string{}
	for _, part := range strings.Split(h, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(k))] = strings.Trim(strings.TrimSpace(v), `"`)
	}
	return out
}

// Authenticate extracts the public key. Precedence: X-Sentry-Auth header,
// sentry_key query param, envelope header dsn. If an envelope DSN is present
// its project must match the path project. Validating the key is up to the caller.
func Authenticate(in AuthInput) (Credentials, error) {
	c := Credentials{ProjectID: in.ProjectID}
	var envDSN *DSN
	if in.EnvelopeDSN != "" {
		d, err := ParseDSN(in.EnvelopeDSN)
		if err == nil {
			envDSN = &d
		} else if in.SentryAuth == "" && in.Query.Get("sentry_key") == "" {
			return Credentials{}, err
		}
	}
	if k := ParseSentryAuth(in.SentryAuth)["sentry_key"]; in.SentryAuth != "" && k != "" {
		c.PublicKey, c.Source = k, "header"
	} else if k := in.Query.Get("sentry_key"); k != "" {
		c.PublicKey, c.Source = k, "query"
	} else if envDSN != nil {
		c.PublicKey, c.Source = envDSN.PublicKey, "envelope"
	} else {
		return Credentials{}, ErrMissingKey
	}
	if envDSN != nil {
		if c.ProjectID == "" {
			c.ProjectID = envDSN.ProjectID
		} else if envDSN.ProjectID != c.ProjectID {
			return Credentials{}, ErrProjectMismatch
		}
	}
	return c, nil
}
