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

package webhook

import (
	"context"
	"net"
	"net/url"

	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// webhookMaxURLLength defines the max allowed length of a webhook URL.
	webhookMaxURLLength = 2048
	// webhookMaxSecretLength defines the max allowed length of a webhook secret.
	webhookMaxSecretLength = 4096
)

var ErrInternalWebhookOperationNotAllowed = errors.Forbidden("changes to internal webhooks are not allowed")

// CheckURL validates the url of a webhook.
func CheckURL(rawURL string, allowLoopback bool, allowPrivateNetwork bool, internal bool) error {
	// for internal webhooks skip URL validation as it is not used
	if internal {
		return nil
	}

	// check URL
	if len(rawURL) > webhookMaxURLLength {
		return check.NewValidationErrorf("The URL of a webhook can be at most %d characters long.",
			webhookMaxURLLength)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return check.NewValidationErrorf("The provided webhook url is invalid: %s", err)
	}

	host := parsedURL.Hostname()
	if host == "" {
		return check.NewValidationError("The URL of a webhook has to have a non-empty host.")
	}

	// basic validation for loopback / private network addresses (only sanitary to give user an early error)
	// IMPORTANT: during webook execution loopback / private network addresses are blocked (handles DNS resolution)

	if host == "localhost" {
		return check.NewValidationError("localhost is not allowed.")
	}

	if ip := net.ParseIP(host); ip != nil {
		if !allowLoopback && ip.IsLoopback() {
			return check.NewValidationError("Loopback IP addresses are not allowed.")
		}

		if !allowPrivateNetwork && ip.IsPrivate() {
			return check.NewValidationError("Private IP addresses are not allowed.")
		}
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return check.NewValidationError("The scheme of a webhook must be either http or https.")
	}

	return nil
}

// CheckSecret validates the secret of a webhook.
func CheckSecret(secret string) error {
	if len(secret) > webhookMaxSecretLength {
		return check.NewValidationErrorf("The secret of a webhook can be at most %d characters long.",
			webhookMaxSecretLength)
	}

	return nil
}

// CheckTriggers validates the triggers of a webhook.
func CheckTriggers(triggers []enum.WebhookTrigger) error {
	// ignore duplicates here, should be deduplicated later
	for _, trigger := range triggers {
		if _, ok := trigger.Sanitize(); !ok {
			return check.NewValidationErrorf("The provided webhook trigger '%s' is invalid.", trigger)
		}
	}

	return nil
}

// DeduplicateTriggers de-duplicates the triggers provided by the user.
func DeduplicateTriggers(in []enum.WebhookTrigger) []enum.WebhookTrigger {
	if len(in) == 0 {
		return []enum.WebhookTrigger{}
	}

	triggerSet := make(map[enum.WebhookTrigger]bool, len(in))
	out := make([]enum.WebhookTrigger, 0, len(in))
	for _, trigger := range in {
		if triggerSet[trigger] {
			continue
		}
		triggerSet[trigger] = true
		out = append(out, trigger)
	}

	return out
}

func ConvertTriggers(vals []string) []enum.WebhookTrigger {
	res := make([]enum.WebhookTrigger, len(vals))
	for i := range vals {
		res[i] = enum.WebhookTrigger(vals[i])
	}
	return res
}

func shouldAuditWebhook(webhookType enum.WebhookType) bool {
	return webhookType == enum.WebhookTypeExternal
}

func getWebhookAuditInfo(parentType enum.WebhookParent) (audit.ResourceType, string) {
	switch parentType {
	case enum.WebhookParentSpace:
		return audit.ResourceTypeCodeWebhook, audit.SpaceName
	case enum.WebhookParentRegistry:
		return audit.ResourceTypeRegistryWebhook, audit.RegistryName
	case enum.WebhookParentRepo:
		return audit.ResourceTypeCodeWebhook, audit.RepoName
	default:
		return "", ""
	}
}

func (s *Service) sendSSE(
	ctx context.Context,
	parentResource ParentResource,
	sseType enum.SSEType,
	webhook *types.Webhook,
) {
	spaceID := parentResource.ID
	if parentResource.Type == enum.WebhookParentRepo {
		repo, err := s.repoStore.Find(ctx, parentResource.ID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to find repo")
			return
		}
		spaceID = repo.ParentID
	}
	s.sseStreamer.Publish(ctx, spaceID, sseType, webhook)
}
