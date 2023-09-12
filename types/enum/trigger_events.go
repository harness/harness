// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// TriggerEvent defines the different kinds of events in triggers.
type TriggerEvent string

// Hook event constants.
const (
	TriggerEventCron        = "cron"
	TriggerEventManual      = "manual"
	TriggerEventPush        = "push"
	TriggerEventPullRequest = "pull_request"
	TriggerEventTag         = "tag"
)
