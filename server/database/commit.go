package database

import (
	"time"

	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

type CommitManager interface {
	// Find finds the commit by ID.
	Find(id int64) (*model.Commit, error)

	// FindSha finds the commit for the branch and sha.
	FindSha(repo int64, branch, sha string) (*model.Commit, error)

	// FindLatest finds the most recent commit for the branch.
	FindLatest(repo int64, branch string) (*model.Commit, error)

	// FindOutput finds the commit's output.
	FindOutput(commit int64) ([]byte, error)

	// List finds recent commits for the repository
	List(repo int64) ([]*model.Commit, error)

	// ListBranch finds recent commits for the repository and branch.
	ListBranch(repo int64, branch string) ([]*model.Commit, error)

	// ListBranches finds most recent commit for each branch.
	ListBranches(repo int64) ([]*model.Commit, error)

	// ListUser finds most recent commits for a user.
	ListUser(repo int64) ([]*model.CommitRepo, error)

	// Insert persists the commit to the datastore.
	Insert(commit *model.Commit) error

	// Update persists changes to the commit to the datastore.
	Update(commit *model.Commit) error

	// UpdateOutput persists a commit's stdout to the datastore.
	UpdateOutput(commit *model.Commit, out []byte) error

	// Delete removes the commit from the datastore.
	Delete(commit *model.Commit) error

	// CancelAll will update the status of all Started or Pending
	// builds to a status of Killed (cancelled).
	CancelAll() error
}

// commitManager manages a list of commits in a SQL database.
type commitManager struct {
	ORM *gorm.DB
}

// NewCommitManager initiales a new CommitManager intended to
// manage and persist commits.
func NewCommitManager(db *gorm.DB) CommitManager {
	return &commitManager{ORM: db}
}

// SQL query to retrieve the latest Commits for each branch.
const listBranchesQuery = `
SELECT *
FROM commits
WHERE id IN (
    SELECT MAX(id)
    FROM commits
    WHERE repo_id=?
    AND status NOT IN ('Started', 'Pending')
    GROUP BY branch)
 ORDER BY branch ASC
 `

// SQL query to retrieve the latest Commits for a user's repositories.
const listUserCommitsQuery = `
SELECT r.remote, r.host, r.owner, r.name, c.*
FROM commits c
	JOIN repos r ON r.id = c.repo_id
	JOIN perms p ON p.repo_id = r.id
	WHERE p.user_id = ?
	AND   c.id IS NOT NULL
	AND   c.status NOT IN ('Started', 'Pending')
	GROUP BY r.id
ORDER BY c.created DESC LIMIT 5
`

func (db *commitManager) Find(id int64) (*model.Commit, error) {
	commit := model.Commit{}

	err := db.ORM.Table("commits").Where(model.Commit{Id: id}).First(&commit).Error
	return &commit, err
}

func (db *commitManager) FindSha(repo int64, branch, sha string) (*model.Commit, error) {
	commit := model.Commit{}

	err := db.ORM.Table("commits").Where(model.Commit{RepoId: repo, Branch: branch, Sha: sha}).First(&commit).Error
	return &commit, err
}

func (db *commitManager) FindLatest(repo int64, branch string) (*model.Commit, error) {
	var max_commit int64
	commit := model.Commit{}

	row := db.ORM.Table("commits").Select("MAX(id)").Where(model.Commit{RepoId: repo, Branch: branch}).Row()
	row.Scan(&max_commit)

	err := db.ORM.Table("commits").Where(max_commit).First(&commit).Error
	return &commit, err
}

func (db *commitManager) FindOutput(commit int64) ([]byte, error) {
	var output string

	row := db.ORM.Table("output").Select("output_raw").Where(model.Output{CommitId: commit}).Row()
	err := row.Scan(&output)

	return []byte(output), err
}

func (db *commitManager) List(repo int64) ([]*model.Commit, error) {
	var commits []*model.Commit

	err := db.ORM.Table("commits").Where(model.Commit{RepoId: repo}).Order("id desc").Find(&commits).Error
	return commits, err
}

func (db *commitManager) ListBranch(repo int64, branch string) ([]*model.Commit, error) {
	var commits []*model.Commit

	err := db.ORM.Table("commits").Where(model.Commit{RepoId: repo, Branch: branch}).Order("id desc").Limit("20").Find(&commits).Error
	return commits, err
}

func (db *commitManager) ListBranches(repo int64) ([]*model.Commit, error) {
	var commits []*model.Commit

	rows, err := db.ORM.Raw(listBranchesQuery, repo).Find(&commits).Rows()
	rows.Scan(&commits)

	return commits, err
}

func (db *commitManager) ListUser(user int64) ([]*model.CommitRepo, error) {
	var commit_repos []*model.CommitRepo

	rows, err := db.ORM.Raw(listUserCommitsQuery, user).Rows()
	rows.Scan(&commit_repos)

	return commit_repos, err
}

func (db *commitManager) Insert(commit *model.Commit) error {
	commit.Created = time.Now().Unix()
	commit.Updated = time.Now().Unix()

	return db.ORM.Table("commits").Create(commit).Error
}

func (db *commitManager) Update(commit *model.Commit) error {
	commit.Updated = time.Now().Unix()

	return db.ORM.Table("commits").Where(model.Commit{Id: commit.Id}).Update(commit).Error
}

func (db *commitManager) UpdateOutput(commit *model.Commit, out []byte) error {
	if err := db.ORM.Table("output").Create(model.Output{CommitId: commit.Id, OutputRaw: string(out)}).Error; err != nil {
		return nil
	}

	output := model.Output{}
	if err := db.ORM.Table("output").Where(model.Output{CommitId: commit.Id}).First(&output).Error; err != nil {
		return err
	}

	if string(out) != output.OutputRaw {
		return db.ORM.Table("output").Where(model.Output{CommitId: commit.Id}).Update(output).Error
	}

	return nil
}

func (db *commitManager) Delete(commit *model.Commit) error {
	return db.ORM.Table("commits").Delete(commit).Error
}

func (db *commitManager) CancelAll() error {
	err := db.ORM.Table("commits").Where("status IN ('Started', 'Pending')").
		Updates(model.Commit{Status: model.StatusKilled, Started: time.Now().Unix(), Finished: time.Now().Unix()}).Error
	return err
}
