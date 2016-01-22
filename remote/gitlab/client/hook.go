package client

import (
	"encoding/json"
	"fmt"
)

// ParseHook parses hook payload from GitLab
func ParseHook(payload []byte) (*HookPayload, error) {
	hp := HookPayload{}
	if err := json.Unmarshal(payload, &hp); err != nil {
		return nil, err
	}

	// Basic sanity check
	switch {
	case len(hp.ObjectKind) == 0:
		// Assume this is a post-receive within repository
		if len(hp.After) == 0 {
			return nil, fmt.Errorf("Invalid hook received, commit hash not found.")
		}
	case hp.ObjectKind == "push":
		if hp.Repository == nil {
			return nil, fmt.Errorf("Invalid push hook received, attributes not found")
		}
	case hp.ObjectKind == "tag_push":
		if hp.Repository == nil {
			return nil, fmt.Errorf("Invalid tag push hook received, attributes not found")
		}
	case hp.ObjectKind == "issue":
		fallthrough
	case hp.ObjectKind == "merge_request":
		if hp.ObjectAttributes == nil {
			return nil, fmt.Errorf("Invalid hook received, attributes not found.")
		}
	default:
		return nil, fmt.Errorf("Invalid hook received, payload format not recognized.")
	}

	return &hp, nil
}
