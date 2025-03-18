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

package metric

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/posthog/posthog-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const postHogGroupInstall = "install"
const postHogServerUserID = "harness-server"

type PostHog struct {
	client             posthog.Client
	installID          string
	hostname           string
	principalStore     store.PrincipalStore
	principalInfoCache store.PrincipalInfoCache
}

type group struct {
	Type       string
	ID         string
	Properties map[string]any
}

func NewPostHog(
	ctx context.Context,
	config *types.Config,
	values *Values,
	principalStore store.PrincipalStore,
	principalInfoCache store.PrincipalInfoCache,
) (*PostHog, error) {
	if !values.Enabled || values.InstallID == "" || config.Metric.PostHogProjectAPIKey == "" {
		return nil, nil //nolint:nilnil // PostHog is disabled
	}

	logr := log.Ctx(ctx).With().Str("service.name", "posthog").Logger()

	// https://posthog.com/docs/libraries/go#overriding-geoip-properties

	phConfig := posthog.Config{
		Endpoint:               config.Metric.PostHogEndpoint,
		PersonalApiKey:         config.Metric.PostHogPersonalAPIKey,
		Logger:                 &logger{Logger: logr},
		DefaultEventProperties: posthog.NewProperties().Set("install_id", values.InstallID),
		Callback:               nil,
	}

	client, err := posthog.NewWithConfig(config.Metric.PostHogProjectAPIKey, phConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostHog client: %w", err)
	}

	ph := &PostHog{
		client:             client,
		installID:          values.InstallID,
		hostname:           values.Hostname,
		principalStore:     principalStore,
		principalInfoCache: principalInfoCache,
	}

	go ph.submitDefaultGroupOnce(ctx)

	return ph, nil
}

func (ph *PostHog) SubmitGroups(context.Context) error {
	// No implementation
	return nil
}

func (ph *PostHog) SubmitForRepo(
	ctx context.Context,
	user *types.PrincipalInfo,
	verb VerbRepo,
	properties map[string]any,
) error {
	return ph.submit(ctx, user, ObjectRepository, string(verb), properties)
}

func (ph *PostHog) SubmitForPullReq(
	ctx context.Context,
	user *types.PrincipalInfo,
	verb VerbPullReq,
	properties map[string]any,
) error {
	return ph.submit(ctx, user, ObjectPullRequest, string(verb), properties)
}

func (ph *PostHog) uniqueUserID(id string) string {
	return ph.installID + ":" + id
}

func (ph *PostHog) submit(
	_ context.Context,
	user *types.PrincipalInfo,
	object Object,
	verb string,
	properties map[string]any,
) error {
	if ph == nil {
		return nil
	}

	var distinctID string
	if user != nil {
		distinctID = ph.uniqueUserID(user.UID)

		p := posthog.NewProperties().Merge(properties)
		p.Set("$set_once", map[string]any{
			"type":    user.Type,
			"created": user.Created,
		})
		p.Set("$set", map[string]any{
			"email": user.Email,
		})

		properties = p
	}

	err := ph.client.Enqueue(posthog.Capture{
		DistinctId: distinctID,
		Event:      string(object) + ":" + verb,
		Groups:     posthog.NewGroups().Set(postHogGroupInstall, ph.installID),
		Properties: properties,
	})
	if err != nil {
		return fmt.Errorf("failed to enqueue event; object=%s verb=%s: %w", object, verb, err)
	}

	return nil
}

func (ph *PostHog) submitGroup(group group) error {
	err := ph.client.Enqueue(posthog.GroupIdentify{
		DistinctId: postHogServerUserID,
		Type:       group.Type,
		Key:        group.ID,
		Properties: group.Properties,
	})
	if err != nil {
		return fmt.Errorf("failed to enqueue group identify: %w", err)
	}

	return nil
}

func (ph *PostHog) submitDefaultGroup(ctx context.Context) error {
	users, err := ph.principalStore.ListUsers(ctx, &types.UserFilter{
		Page:  1,
		Size:  1,
		Sort:  enum.UserAttrCreated,
		Order: enum.OrderAsc,
	})
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	if len(users) == 0 {
		return errors.New("no users found")
	}

	userFirst := users[0]

	// Note: The PostHog UI identifies a group using the name property.
	// If the name property is not found, it falls back to the group key.
	// https://posthog.com/docs/product-analytics/group-analytics#how-to-set-group-properties
	g := group{
		Type: postHogGroupInstall,
		ID:   ph.installID,
		Properties: posthog.NewProperties().
			Set("name", "install").
			Set("hostname", ph.hostname).
			Set("email", userFirst.Email).
			Set("created", userFirst.Created),
	}

	err = ph.submitGroup(g)
	if err != nil {
		return fmt.Errorf("failed to submit default group: %w", err)
	}

	return nil
}

func (ph *PostHog) submitDefaultGroupOnce(ctx context.Context) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	logr := log.Ctx(ctx).With().Str("service.name", "posthog").Logger()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := ph.submitDefaultGroup(ctx); err != nil {
				logr.Err(err).Msg("failed to submit default group")
				timer.Reset(time.Hour)
				continue
			}

			return
		}
	}
}

type logger struct {
	zerolog.Logger
}

func (l *logger) Logf(format string, args ...interface{}) {
	l.Info().Msgf(format, args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.Error().Msgf(format, args...)
}
