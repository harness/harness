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

package job

import "time"

type Config struct {
	// InstanceID specifis the ID of the instance.
	InstanceID string `envconfig:"INSTANCE_ID"`

	// MaxRunning is maximum number of jobs that can be running at once.
	BackgroundJobsMaxRunning int `envconfig:"JOBS_MAX_RUNNING" default:"10"`

	// RetentionTime is the duration after which non-recurring,
	// finished and failed jobs will be purged from the DB.
	BackgroundJobsRetentionTime time.Duration `envconfig:"JOBS_RETENTION_TIME" default:"120h"` // 5 days

}
