package client

type QMap map[string]string

type User struct {
	Id        int    `json:"id,omitempty"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	AvatarUrl string `json:"avatar_url,omitempty"`
	Name      string `json:"name,omitempty"`
}

type ProjectAccess struct {
	AccessLevel       int `json:"access_level,omitempty"`
	NotificationLevel int `json:"notification_level,omitempty"`
}

type GroupAccess struct {
	AccessLevel       int `json:"access_level,omitempty"`
	NotificationLevel int `json:"notification_level,omitempty"`
}

type Permissions struct {
	ProjectAccess *ProjectAccess `json:"project_access,omitempty"`
	GroupAccess   *GroupAccess   `json:"group_access,omitempty"`
}

type Member struct {
	Id        int
	Username  string
	Email     string
	Name      string
	State     string
	CreatedAt string `json:"created_at,omitempty"`
	// AccessLevel int
}

type Project struct {
	Id                int          `json:"id,omitempty"`
	Owner             *Member      `json:"owner,omitempty"`
	Name              string       `json:"name,omitempty"`
	Description       string       `json:"description,omitempty"`
	DefaultBranch     string       `json:"default_branch,omitempty"`
	Public            bool         `json:"public,omitempty"`
	Path              string       `json:"path,omitempty"`
	PathWithNamespace string       `json:"path_with_namespace,omitempty"`
	Namespace         *Namespace   `json:"namespace,omitempty"`
	SshRepoUrl        string       `json:"ssh_url_to_repo"`
	HttpRepoUrl       string       `json:"http_url_to_repo"`
	Url               string       `json:"web_url"`
	AvatarUrl         string       `json:"avatar_url"`
	Permissions       *Permissions `json:"permissions,omitempty"`
}

type Namespace struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
}

type Person struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type hProject struct {
	Name              string `json:"name"`
	SshUrl            string `json:"ssh_url"`
	HttpUrl           string `json:"http_url"`
	GitSshUrl         string `json:"git_ssh_url"`
	GitHttpUrl        string `json:"git_http_url"`
	AvatarUrl         string `json:"avatar_url"`
	VisibilityLevel   int    `json:"visibility_level"`
	WebUrl            string `json:"web_url"`
	PathWithNamespace string `json:"path_with_namespace"`
	DefaultBranch     string `json:"default_branch"`
	Namespace         string `json:"namespace"`
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

type HookPayload struct {
	Before            string       `json:"before,omitempty"`
	After             string       `json:"after,omitempty"`
	Ref               string       `json:"ref,omitempty"`
	UserId            int          `json:"user_id,omitempty"`
	UserName          string       `json:"user_name,omitempty"`
	ProjectId         int          `json:"project_id,omitempty"`
	Project           *hProject    `json:"project,omitempty"`
	Repository        *hRepository `json:"repository,omitempty"`
	Commits           []hCommit    `json:"commits,omitempty"`
	TotalCommitsCount int          `json:"total_commits_count,omitempty"`
	ObjectKind        string       `json:"object_kind,omitempty"`
	ObjectAttributes  *HookObjAttr `json:"object_attributes,omitempty"`
}
