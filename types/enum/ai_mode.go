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

package enum

// AIMode defines how much repository data may be sent to AI models.
//
// Spanish names used in the design doc: full = "completo", decisions-only = "solo-decisiones",
// off = "apagado".
type AIMode string

func (AIMode) Enum() []any {
	return toInterfaceSlice(AIModes)
}

var AIModes = sortEnum([]AIMode{
	AIModeFull,
	AIModeDecisionsOnly,
	AIModeOff,
})

const (
	// AIModeFull ("completo"): redacted content including code may be sent to generative models;
	// fix agents are allowed.
	AIModeFull AIMode = "full"
	// AIModeDecisionsOnly ("solo-decisiones"): only compact metadata goes to the decision model;
	// no code or payloads to generative models; no fix agents.
	AIModeDecisionsOnly AIMode = "decisions-only"
	// AIModeOff ("apagado"): nothing is sent.
	AIModeOff AIMode = "off"
)

func (m AIMode) Sanitize() (AIMode, bool) {
	return Sanitize(m, GetAllAIModes)
}

func GetAllAIModes() ([]AIMode, AIMode) {
	return AIModes, ""
}
