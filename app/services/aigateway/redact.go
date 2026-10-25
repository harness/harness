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

package aigateway

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/zricethezav/gitleaks/v8/detect"
)

type redactRule struct {
	kind string
	re   *regexp.Regexp
	// group is the submatch that holds the sensitive value; 0 means the whole match.
	group int
}

var redactRules = []redactRule{
	{kind: "private_key", re: regexp.MustCompile(
		`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)},
	{kind: "jwt", re: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`)},
	{kind: "auth_header", re: regexp.MustCompile(`(?i)\b(?:authorization|proxy-authorization)\s*[:=]\s*([^\r\n"']+)`), group: 1},
	{kind: "bearer", re: regexp.MustCompile(`(?i)\bbearer\s+([A-Za-z0-9._~+/=-]{8,})`), group: 1},
	{kind: "cookie", re: regexp.MustCompile(`(?i)\b(?:set-cookie|cookie)\s*[:=]\s*([^\r\n"']+)`), group: 1},
	{kind: "credential", re: regexp.MustCompile(
		// no leading \b so that prefixed names such as DB_PASSWORD or OPENROUTER_API_KEY match
		`(?i)(?:password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key|client[_-]?secret)\b` +
			`["']?\s*[:=]\s*["']?([^\s"',;&]{4,})`), group: 1},
	{kind: "url_credentials", re: regexp.MustCompile(`\b[a-z][a-z0-9+.-]*://[^\s:/@"']+:([^\s@/"']+)@`), group: 1},
	{kind: "email", re: regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)},
	{kind: "ipv4", re: regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4]\d|1?\d?\d)\.){3}(?:25[0-5]|2[0-4]\d|1?\d?\d)\b`)},
	{kind: "ipv6", re: regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`)},
}

// Redactor removes secrets and personal data before anything is sent to a free provider,
// since free providers may log or train on prompts.
type Redactor struct {
	detector *detect.Detector
}

// NewRedactor combines gitleaks (the same detector used by secret scanning) with
// rules for personal data and auth material that gitleaks doesn't cover.
func NewRedactor() (*Redactor, error) {
	detector, err := detect.NewDetectorDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create gitleaks detector: %w", err)
	}
	return &Redactor{detector: detector}, nil
}

// Redact returns the text with every sensitive value replaced by [REDACTED:<kind>].
func (r *Redactor) Redact(text string) string {
	if text == "" {
		return text
	}

	if r.detector != nil {
		secrets := make([]string, 0)
		for _, f := range r.detector.DetectString(text) {
			if f.Secret != "" {
				secrets = append(secrets, f.Secret)
			}
		}
		// replace longer secrets first so a secret containing another is fully removed
		sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
		for _, s := range secrets {
			text = strings.ReplaceAll(text, s, "[REDACTED:secret]")
		}
	}

	for _, rule := range redactRules {
		text = redactWithRule(text, rule)
	}
	return text
}

func redactWithRule(text string, rule redactRule) string {
	placeholder := "[REDACTED:" + rule.kind + "]"
	if rule.group == 0 {
		return rule.re.ReplaceAllString(text, placeholder)
	}

	matches := rule.re.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var b strings.Builder
	last := 0
	for _, m := range matches {
		start, end := m[2*rule.group], m[2*rule.group+1]
		if start < 0 {
			continue
		}
		b.WriteString(text[last:start])
		b.WriteString(placeholder)
		last = end
	}
	b.WriteString(text[last:])
	return b.String()
}
