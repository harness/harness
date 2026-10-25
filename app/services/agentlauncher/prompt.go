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

package agentlauncher

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

// PromptVersion is the version of the embedded prompt templates (prompts/<kind>.v<version>.md).
const PromptVersion = 1

//go:embed prompts/*.md
var promptFS embed.FS

type promptData struct {
	Number      int64
	Title       string
	Priority    string
	Description string
}

func promptFor(kind Kind) (*template.Template, error) {
	switch kind {
	case KindFixError, KindFixDependency, KindFixSAST:
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}
	name := fmt.Sprintf("prompts/%s.v%d.md", kind, PromptVersion)
	raw, err := promptFS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt %s: %w", name, err)
	}
	// text/template: the prompt is plain text for a model, not HTML
	t, err := template.New(name).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt %s: %w", name, err)
	}
	return t, nil
}

func renderPrompt(kind Kind, data promptData) (string, error) {
	t, err := promptFor(kind)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render prompt: %w", err)
	}
	return buf.String(), nil
}
