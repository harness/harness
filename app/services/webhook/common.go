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
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/netpolicy"
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
	// webhookMaxExtraHeaders defines the max number of custom headers allowed per webhook.
	webhookMaxExtraHeaders = 20
)

var ErrInternalWebhookOperationNotAllowed = errors.Forbidden("changes to internal webhooks are not allowed")

// CheckURL validates the url of a webhook.
func CheckURL(
	rawURL string,
	allowLoopback bool,
	allowPrivateNetwork bool,
	allowLinkLocal bool,
	internal bool,
) error {
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
		switch netpolicy.Classify(ip) {
		case netpolicy.ClassLoopback:
			if !allowLoopback {
				return check.NewValidationError("Loopback IP addresses are not allowed.")
			}
		case netpolicy.ClassLinkLocal:
			if !allowLinkLocal {
				return check.NewValidationError("Link-local IP addresses are not allowed.")
			}
		case netpolicy.ClassPrivate:
			if !allowPrivateNetwork {
				return check.NewValidationError("Private IP addresses are not allowed.")
			}
		case netpolicy.ClassPublic:
			// allowed
		case netpolicy.ClassReserved:
			// unspecified (0.0.0.0/::), broadcast, multicast, reserved ranges
			return check.NewValidationError("IP address is not allowed.")
		default:
			return check.NewValidationError("IP address is not allowed.")
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

// encryptSecret encrypts the webhook secret. An empty secret stays empty, because encrypting an empty string
// produces a non-empty ciphertext, which would make the webhook look like it has a secret (has_secret).
func (s *Service) encryptSecret(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}

	encryptedSecret, err := s.encrypter.Encrypt(secret)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt webhook secret: %w", err)
	}

	return string(encryptedSecret), nil
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

// CheckExtraHeaders validates the custom headers of a webhook.
func CheckExtraHeaders(headers []types.ExtraHeader) error {
	if len(headers) > webhookMaxExtraHeaders {
		return check.NewValidationErrorf("A webhook can have at most %d custom headers.", webhookMaxExtraHeaders)
	}

	headerKeys := make(map[string]struct{}, len(headers))
	for _, header := range headers {
		if header.Key == "" {
			return check.NewValidationError("Header key cannot be empty.")
		}
		// check for duplicate header keys (case insensitive)
		lowerKey := strings.ToLower(header.Key)
		if _, ok := headerKeys[lowerKey]; ok {
			return check.NewValidationErrorf("Duplicate header key '%s' detected.", header.Key)
		}
		headerKeys[lowerKey] = struct{}{}

		if header.Masked && header.Value == "" {
			return check.NewValidationErrorf("Masked header '%s' requires a value.", header.Key)
		}
	}

	return nil
}

// MergeExtraHeaders computes the new set of extra headers of a webhook on update.
// The provided headers replace the existing ones, with one exception: masked header values are never returned
// to clients, so a masked header that is provided without a value keeps the stored value of the existing header
// with the same key (case insensitive). Existing headers that aren't provided are removed.
func MergeExtraHeaders(existing, in []types.ExtraHeader) []types.ExtraHeader {
	existingValues := make(map[string]string, len(existing))
	for _, h := range existing {
		existingValues[strings.ToLower(h.Key)] = h.Value
	}

	out := make([]types.ExtraHeader, len(in))
	for i, h := range in {
		out[i] = h
		if h.Masked && h.Value == "" {
			out[i].Value = existingValues[strings.ToLower(h.Key)]
		}
	}

	return out
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
