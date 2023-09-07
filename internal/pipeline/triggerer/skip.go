// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package triggerer

import (
	"strings"

	"github.com/harness/gitness/types/enum"

	"github.com/drone/drone-yaml/yaml"
)

func skipBranch(document *yaml.Pipeline, branch string) bool {
	return !document.Trigger.Branch.Match(branch)
}

func skipRef(document *yaml.Pipeline, ref string) bool {
	return !document.Trigger.Ref.Match(ref)
}

func skipEvent(document *yaml.Pipeline, event string) bool {
	return !document.Trigger.Event.Match(event)
}

func skipAction(document *yaml.Pipeline, action string) bool {
	return !document.Trigger.Action.Match(action)
}

func skipInstance(document *yaml.Pipeline, instance string) bool {
	return !document.Trigger.Instance.Match(instance)
}

func skipTarget(document *yaml.Pipeline, env string) bool {
	return !document.Trigger.Target.Match(env)
}

func skipRepo(document *yaml.Pipeline, repo string) bool {
	return !document.Trigger.Repo.Match(repo)
}

func skipCron(document *yaml.Pipeline, cron string) bool {
	return !document.Trigger.Cron.Match(cron)
}

func skipMessage(hook *Hook) bool {
	switch {
	case hook.Event == enum.TriggerEventTag:
		return false
	case hook.Event == enum.TriggerEventCron:
		return false
	case hook.Event == enum.TriggerEventCustom:
		return false
	case hook.Event == enum.TriggerEventPromote:
		return false
	case hook.Event == enum.TriggerEventRollback:
		return false
	case skipMessageEval(hook.Message):
		return true
	case skipMessageEval(hook.Title):
		return true
	default:
		return false
	}
}

func skipMessageEval(str string) bool {
	lower := strings.ToLower(str)
	switch {
	case strings.Contains(lower, "[ci skip]"),
		strings.Contains(lower, "[skip ci]"),
		strings.Contains(lower, "***no_ci***"):
		return true
	default:
		return false
	}
}
