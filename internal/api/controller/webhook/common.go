// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"net"
	"net/url"

	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

const (
	// webhookMaxURLLength defines the max allowed length of a webhook URL.
	webhookMaxURLLength = 2048
	// webhookMaxSecretLength defines the max allowed length of a webhook secret.
	webhookMaxSecretLength = 4096
)

// checkURL validates the url of a webhook.
func checkURL(rawURL string, allowLoopback bool, allowPrivateNetwork bool) error {
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

// checkSecret validates the secret of a webhook.
func checkSecret(secret string) error {
	if len(secret) > webhookMaxSecretLength {
		return check.NewValidationErrorf("The secret of a webhook can be at most %d characters long.",
			webhookMaxSecretLength)
	}

	return nil
}

// checkTriggers validates the triggers of a webhook.
func checkTriggers(triggers []enum.WebhookTrigger) error {
	// ignore duplicates here, should be deduplicated later
	for _, trigger := range triggers {
		if _, ok := enum.ParseWebhookTrigger(string(trigger)); !ok {
			return check.NewValidationErrorf("The provided webhook trigger '%s' is invalid.", trigger)
		}
	}

	return nil
}

// deduplicateTriggers de-duplicates the triggers provided by the user.
func deduplicateTriggers(in []enum.WebhookTrigger) []enum.WebhookTrigger {
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
