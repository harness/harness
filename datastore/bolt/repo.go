package bolt

import (
	"time"

	"github.com/drone/drone/common"
)

// GetRepo gets the repository by name.
func (db *DB) GetRepo(repo string) (*common.Repo, error) {
	repo_ := &common.Repo{}
	key := []byte(repo)
	err := get(db, bucketRepo, key, repo_)
	return repo_, err
}

// GetRepoParams gets the private environment parameters
// for the given repository.
func (db *DB) GetRepoParams(repo string) (map[string]string, error) {
	params := map[string]string{}
	key := []byte(repo)
	err := get(db, bucketRepoParams, key, &params)
	return params, err
}

// GetRepoParams gets the private and public rsa keys
// for the given repository.
func (db *DB) GetRepoKeys(repo string) (*common.Keypair, error) {
	keypair := &common.Keypair{}
	key := []byte(repo)
	err := get(db, bucketRepoKeys, key, keypair)
	return keypair, err
}

// UpdateRepos updates a repository. If the repository
// does not exist an error is returned.
func (db *DB) UpdateRepo(repo *common.Repo) error {
	key := []byte(repo.FullName)
	repo.Updated = time.Now().UTC().Unix()
	return update(db, bucketRepo, key, repo)
}

// InsertRepo inserts a repository in the datastore and
// subscribes the user to that repository.
func (db *DB) InsertRepo(user *common.User, repo *common.Repo) error {
	key := []byte(repo.FullName)
	repo.Created = time.Now().UTC().Unix()
	repo.Updated = time.Now().UTC().Unix()
	// TODO(bradrydzewski) add repo to user index
	// TODO(bradrydzewski) add user to repo index
	return insert(db, bucketRepo, key, repo)
}

// UpsertRepoParams inserts or updates the private
// environment parameters for the named repository.
func (db *DB) UpsertRepoParams(repo string, params map[string]string) error {
	key := []byte(repo)
	return update(db, bucketRepoParams, key, params)
}

// UpsertRepoKeys inserts or updates the private and
// public keypair for the named repository.
func (db *DB) UpsertRepoKeys(repo string, keypair *common.Keypair) error {
	key := []byte(repo)
	return update(db, bucketRepoKeys, key, keypair)
}

// DeleteRepo deletes the repository.
func (db *DB) DeleteRepo(repo *common.Repo) error {
	t, err := db.Begin(true)
	if err != nil {
		return err
	}
	key := []byte(repo.FullName)
	err = t.Bucket(bucketRepo).Delete(key)
	if err != nil {
		t.Rollback()
		return err
	}
	t.Bucket(bucketRepoKeys).Delete(key)
	t.Bucket(bucketRepoParams).Delete(key)
	// TODO(bradrydzewski) delete all builds
	// TODO(bradrydzewski) delete all tasks
	return t.Commit()
}

// GetSubscriber gets the subscriber by login for the
// named repository.
func (db *DB) GetSubscriber(repo string, login string) (*common.Subscriber, error) {
	sub := &common.Subscriber{}
	key := []byte(login + "/" + repo)
	err := get(db, bucketUserRepos, key, sub)
	return sub, err
}

// InsertSubscriber inserts a subscriber for the named
// repository.
func (db *DB) InsertSubscriber(repo string, sub *common.Subscriber) error {
	key := []byte(sub.Login + "/" + repo)
	return insert(db, bucketUserRepos, key, sub)
}

// DeleteSubscriber removes the subscriber by login for the
// named repository.
func (db *DB) DeleteSubscriber(repo string, sub *common.Subscriber) error {
	key := []byte(sub.Login + "/" + repo)
	return delete(db, bucketUserRepos, key)
}
