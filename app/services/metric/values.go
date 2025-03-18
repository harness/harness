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

	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/types"

	"github.com/google/uuid"
)

type Values struct {
	Enabled   bool
	Hostname  string
	InstallID string
}

func NewValues(
	ctx context.Context,
	config *types.Config,
	settingsSrv *settings.Service,
) (*Values, error) {
	if !config.Metric.Enabled {
		return &Values{
			Enabled:   false,
			InstallID: "",
		}, nil
	}

	values := Values{
		Enabled:   true,
		Hostname:  config.InstanceID,
		InstallID: "",
	}

	ok, err := settingsSrv.SystemGet(ctx, settings.KeyInstallID, &values.InstallID)
	if err != nil {
		return nil, fmt.Errorf("failed to find install id: %w", err)
	}

	if !ok || values.InstallID == "" {
		values.InstallID = uuid.New().String()
		err = settingsSrv.SystemSet(ctx, settings.KeyInstallID, values.InstallID)
		if err != nil {
			return nil, fmt.Errorf("failed to update system settings: %w", err)
		}
	}

	return &values, nil
}
