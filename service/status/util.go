// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package status

import (
	"fmt"

	"github.com/drone/drone/core"
	"github.com/drone/go-scm/scm"
	"github.com/gosimple/slug"
)

func createLabel(name, event, deployTo string) string {
	if name == "" {
		name = "continuous-integration/drone"
	}
	switch event {
	case core.EventPush:
		return fmt.Sprintf("%s/push", name)
	case core.EventPullRequest:
		return fmt.Sprintf("%s/pr", name)
	case core.EventTag:
		return fmt.Sprintf("%s/tag", name)
	case core.EventPromote:
		return fmt.Sprintf("%s/promote/%s", name, slug.Make(deployTo))
	default:
		return name
	}
}

func createDesc(state string) string {
	switch state {
	case core.StatusBlocked:
		return "Build is pending approval"
	case core.StatusDeclined:
		return "Build was declined"
	case core.StatusError:
		return "Build encountered an error"
	case core.StatusFailing:
		return "Build is failing"
	case core.StatusKilled:
		return "Build was killed"
	case core.StatusPassing:
		return "Build is passing"
	case core.StatusWaiting:
		return "Build is pending"
	case core.StatusPending:
		return "Build is pending"
	case core.StatusRunning:
		return "Build is running"
	case core.StatusSkipped:
		return "Build was skipped"
	default:
		return "Build is in an unknown state"
	}
}

func convertStatus(state string) scm.State {
	switch state {
	case core.StatusBlocked:
		return scm.StatePending
	case core.StatusDeclined:
		return scm.StateCanceled
	case core.StatusError:
		return scm.StateError
	case core.StatusFailing:
		return scm.StateFailure
	case core.StatusKilled:
		return scm.StateCanceled
	case core.StatusPassing:
		return scm.StateSuccess
	case core.StatusPending:
		return scm.StatePending
	case core.StatusRunning:
		return scm.StatePending
	case core.StatusSkipped:
		return scm.StateUnknown
	default:
		return scm.StateUnknown
	}
}
