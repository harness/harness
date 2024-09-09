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

package messaging

import (
	"os"

	"github.com/slack-go/slack"
)

const (
	SlackBotToken = "HARNESS_INTELLIGENCE_SLACK_BOT_TOKEN"
)

type Slack struct {
	Client     *slack.Client
	SlackBotID string
}

func NewSlack() *Slack {
	token := os.Getenv(SlackBotToken)
	api := slack.New(token)

	return &Slack{
		Client: api,
	}
}
