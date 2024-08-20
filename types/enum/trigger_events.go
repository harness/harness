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

// TriggerEvent defines the different kinds of events in triggers.
type TriggerEvent string

// Hook event constants.
const (
	TriggerEventCron        TriggerEvent = "cron"
	TriggerEventManual      TriggerEvent = "manual"
	TriggerEventPush        TriggerEvent = "push"
	TriggerEventPullRequest TriggerEvent = "pull_request"
	TriggerEventTag         TriggerEvent = "tag"
)

// Enum returns all possible TriggerEvent values.
func (TriggerEvent) Enum() []interface{} {
	return toInterfaceSlice(triggerEvents)
}

// Sanitize validates and returns a sanitized TriggerEvent value.
func (event TriggerEvent) Sanitize() (TriggerEvent, bool) {
	return Sanitize(event, GetAllTriggerEvents)
}

// GetAllTriggerEvents returns all possible TriggerEvent values and a default value.
func GetAllTriggerEvents() ([]TriggerEvent, TriggerEvent) {
	return triggerEvents, TriggerEventManual
}

// List of all TriggerEvent values.
var triggerEvents = sortEnum([]TriggerEvent{
	TriggerEventCron,
	TriggerEventManual,
	TriggerEventPush,
	TriggerEventPullRequest,
	TriggerEventTag,
})
