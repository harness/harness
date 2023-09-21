package enum

// TriggerAction defines the different actions on triggers will fire.
type TriggerAction string

// These are similar to enums defined in webhook enum but can diverge
// as these are different entities.
const (
	// TriggerActionBranchCreated gets triggered when a branch gets created.
	TriggerActionBranchCreated TriggerAction = "branch_created"
	// TriggerActionBranchUpdated gets triggered when a branch gets updated.
	TriggerActionBranchUpdated TriggerAction = "branch_updated"

	// TriggerActionTagCreated gets triggered when a tag gets created.
	TriggerActionTagCreated TriggerAction = "tag_created"
	// TriggerActionTagUpdated gets triggered when a tag gets updated.
	TriggerActionTagUpdated TriggerAction = "tag_updated"

	// TriggerActionPullReqCreated gets triggered when a pull request gets created.
	TriggerActionPullReqCreated TriggerAction = "pullreq_created"
	// TriggerActionPullReqReopened gets triggered when a pull request gets reopened.
	TriggerActionPullReqReopened TriggerAction = "pullreq_reopened"
	// TriggerActionPullReqBranchUpdated gets triggered when a pull request source branch gets updated.
	TriggerActionPullReqBranchUpdated TriggerAction = "pullreq_branch_updated"
)

func (TriggerAction) Enum() []interface{}               { return toInterfaceSlice(triggerActions) }
func (s TriggerAction) Sanitize() (TriggerAction, bool) { return Sanitize(s, GetAllTriggerActions) }
func (t TriggerAction) GetTriggerEvent() TriggerEvent {
	if t == TriggerActionPullReqCreated ||
		t == TriggerActionPullReqBranchUpdated ||
		t == TriggerActionPullReqReopened {
		return TriggerEventPullRequest
	} else if t == TriggerActionTagCreated || t == TriggerActionTagUpdated {
		return TriggerEventTag
	} else if t == "" {
		return TriggerEventManual
	}
	return TriggerEventPush
}

func GetAllTriggerActions() ([]TriggerAction, TriggerAction) {
	return triggerActions, "" // No default value
}

var triggerActions = sortEnum([]TriggerAction{
	TriggerActionBranchCreated,
	TriggerActionBranchUpdated,
	TriggerActionTagCreated,
	TriggerActionTagUpdated,
	TriggerActionPullReqCreated,
	TriggerActionPullReqReopened,
	TriggerActionPullReqBranchUpdated,
})

// Trigger types
const (
	TriggerHook = "@hook"
	TriggerCron = "@cron"
)
