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

package aiagent

import (
	"context"
	"fmt"

	"github.com/slack-go/slack/slackevents"
)

type SlackbotOutput struct {
	Success bool
}

func (c *Controller) HandleEvent(
	_ context.Context,
	eventsAPIEvent slackevents.EventsAPIEvent) (*SlackbotOutput, error) {
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		success, err := c.HandleCallbackEvent(eventsAPIEvent.InnerEvent)
		if err != nil {
			return nil, err
		}
		return &SlackbotOutput{Success: success}, nil
	}
	return nil, fmt.Errorf("unknown event type: %s", eventsAPIEvent.Type)
}

func (c *Controller) HandleCallbackEvent(innerEvent slackevents.EventsAPIInnerEvent) (bool, error) {
	switch innerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
	default:
		// no action needed for unhandled event types
	}
	return true, nil
}
