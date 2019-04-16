// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package webhook

import (
	"context"
	"net/http"
	"testing"

	"github.com/drone/drone/core"

	"github.com/99designs/httpsignatures-go"
	"github.com/h2non/gock"
)

var noContext = context.Background()

func TestWebhook(t *testing.T) {
	defer gock.Off()

	webhook := &core.WebhookData{
		Event:  core.WebhookEventUser,
		Action: core.WebhookActionCreated,
		User:   &core.User{Login: "octocat"},
	}

	matchSignature := func(r *http.Request, _ *gock.Request) (bool, error) {
		signature, err := httpsignatures.FromRequest(r)
		if err != nil {
			return false, err
		}
		return signature.IsValid("GMEuUHQfmrMRsseWxi9YlIeBtn9lm6im", r), nil
	}

	gock.New("https://company.com").
		Post("/hooks").
		AddMatcher(matchSignature).
		MatchHeader("X-Drone-Event", "user").
		MatchHeader("Content-Type", "application/json").
		MatchHeader("Digest", "SHA-256=Rro8edtwJUjLl6e\\/xB9m1ykaymiaC9YxFGb\\/XNuXzO4=").
		JSON(webhook).
		Reply(200).
		Type("application/json")

	config := Config{
		Endpoint: []string{"https://company.com/hooks"},
		Secret:   "GMEuUHQfmrMRsseWxi9YlIeBtn9lm6im",
	}
	sender := New(config)
	err := sender.Send(noContext, webhook)
	if err != nil {
		t.Error(err)
	}

	if gock.IsPending() {
		t.Errorf("Unfinished requests")
	}
}

func TestWebhook_CustomClient(t *testing.T) {
	sender := new(sender)
	if sender.client() != http.DefaultClient {
		t.Errorf("Expect default http client")
	}

	custom := &http.Client{}
	sender.Client = custom
	if sender.client() != custom {
		t.Errorf("Expect custom http client")
	}
}

func TestWebhook_NoEndpoints(t *testing.T) {
	webhook := &core.WebhookData{
		Event:  core.WebhookEventUser,
		Action: core.WebhookActionCreated,
		User:   &core.User{Login: "octocat"},
	}

	config := Config{
		Endpoint: []string{},
		Secret:   "correct-horse-battery-staple",
	}
	sender := New(config)
	err := sender.Send(noContext, webhook)
	if err != nil {
		t.Error(err)
	}
}
