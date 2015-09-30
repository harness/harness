package gogitlab

import (
	"encoding/json"
	"fmt"
	"strings"
)

type HookObjAttr struct {
	Id              int       `json:"id,omitempty"`
	Title           string    `json:"title,omitempty"`
	AssigneeId      int       `json:"assignee_id,omitempty"`
	AuthorId        int       `json:"author_id,omitempty"`
	ProjectId       int       `json:"project_id,omitempty"`
	CreatedAt       string    `json:"created_at,omitempty"`
	UpdatedAt       string    `json:"updated_at,omitempty"`
	Position        int       `json:"position,omitempty"`
	BranchName      string    `json:"branch_name,omitempty"`
	Description     string    `json:"description,omitempty"`
	MilestoneId     int       `json:"milestone_id,omitempty"`
	State           string    `json:"state,omitempty"`
	IId             int       `json:"iid,omitempty"`
	TargetBranch    string    `json:"target_branch,omitempty"`
	SourceBranch    string    `json:"source_branch,omitempty"`
	SourceProjectId int       `json:"source_project_id,omitempty"`
	StCommits       string    `json:"st_commits,omitempty"`
	StDiffs         string    `json:"st_diffs,omitempty"`
	MergeStatus     string    `json:"merge_status,omitempty"`
	TargetProjectId int       `json:"target_project_id,omitempty"`
	Url             string    `json:"url,omiyempty"`
	Source          *hProject `json:"source,omitempty"`
	Target          *hProject `json:"target,omitempty"`
	LastCommit      *hCommit  `json:"last_commit,omitempty"`
}

type hProject struct {
	Name            string `json:"name"`
	SshUrl          string `json:"ssh_url"`
	HttpUrl         string `json:"http_url"`
	VisibilityLevel int    `json:"visibility_level"`
	WebUrl          string `json:"web_url"`
	Namespace       string `json:"namespace"`
}

type hRepository struct {
	Name            string `json:"name,omitempty"`
	URL             string `json:"url,omitempty"`
	Description     string `json:"description,omitempty"`
	Homepage        string `json:"homepage,omitempty"`
	GitHttpUrl      string `json:"git_http_url,omitempty"`
	GitSshUrl       string `json:"git_ssh_url,omitempty"`
	VisibilityLevel int    `json:"visibility_level,omitempty"`
}

type hCommit struct {
	Id        string  `json:"id,omitempty"`
	Message   string  `json:"message,omitempty"`
	Timestamp string  `json:"timestamp,omitempty"`
	URL       string  `json:"url,omitempty"`
	Author    *Person `json:"author,omitempty"`
}

type HookPayload struct {
	Before            string       `json:"before,omitempty"`
	After             string       `json:"after,omitempty"`
	Ref               string       `json:"ref,omitempty"`
	UserId            int          `json:"user_id,omitempty"`
	UserName          string       `json:"user_name,omitempty"`
	ProjectId         int          `json:"project_id,omitempty"`
	Repository        *hRepository `json:"repository,omitempty"`
	Commits           []hCommit    `json:"commits,omitempty"`
	TotalCommitsCount int          `json:"total_commits_count,omitempty"`
	ObjectKind        string       `json:"object_kind,omitempty"`
	ObjectAttributes  *HookObjAttr `json:"object_attributes,omitempty"`
}

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

// Type return current event type
// This function returns "unknown" type if event not supported
func (h *HookPayload) Type() string {
	switch {
	case strings.HasPrefix(h.Ref, "refs/heads/") && len(h.After) == 0:
		return "branch_deleted"
	case strings.HasPrefix(h.Ref, "refs/heads/") && len(h.Before) == 0:
		return "branch"
	case strings.HasPrefix(h.Ref, "refs/heads/"):
		return "commit"
	case strings.HasPrefix(h.Ref, "refs/tags/") && len(h.After) == 0:
		return "tag_deleted"
	case strings.HasPrefix(h.Ref, "refs/tags/"):
		return "tag"
	case h.ObjectKind == "issue":
		return "issue"
	case h.ObjectKind == "merge_request":
		return "merge_request"
	default:
		return "unknown"
	}
}

// Tag returns current tag for push event hook payload
// This function returns empty string for any other events
func (h *HookPayload) Tag() string {
	return strings.TrimPrefix(h.Ref, "refs/tags/")
}

// Branch returns current branch for push event hook payload
// This function returns empty string for any other events
func (h *HookPayload) Branch() string {
	return strings.TrimPrefix(h.Ref, "refs/heads/")
}

// Head returns the latest changeset for push event hook payload
func (h *HookPayload) Head() hCommit {
	c := hCommit{}
	for _, cm := range h.Commits {
		if h.After == cm.Id {
			return cm
		}
	}
	return c
}
