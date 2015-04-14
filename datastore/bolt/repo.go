package bolt

import (
	"bytes"
	"time"

	"github.com/boltdb/bolt"
	"github.com/drone/drone/common"
)

// GetRepo gets the repository by name.
func (db *DB) GetRepo(repo string) (*common.Repo, error) {
	repo_ := &common.Repo{}
	key := []byte(repo)

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketRepo, key, repo_)
	})

	return repo_, err
}

// GetRepoParams gets the private environment parameters
// for the given repository.
func (db *DB) GetRepoParams(repo string) (map[string]string, error) {
	params := map[string]string{}
	key := []byte(repo)

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketRepoParams, key, &params)
	})

	return params, err
}

// GetRepoParams gets the private and public rsa keys
// for the given repository.
func (db *DB) GetRepoKeys(repo string) (*common.Keypair, error) {
	keypair := &common.Keypair{}
	key := []byte(repo)

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketRepoKeys, key, keypair)
	})

	return keypair, err
}

// UpdateRepos updates a repository. If the repository
// does not exist an error is returned.
func (db *DB) UpdateRepo(repo *common.Repo) error {
	key := []byte(repo.FullName)
	repo.Updated = time.Now().UTC().Unix()

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketRepo, key, repo)
	})
}

// InsertRepo inserts a repository in the datastore and
// subscribes the user to that repository.
func (db *DB) InsertRepo(user *common.User, repo *common.Repo) error {
	repokey := []byte(repo.FullName)
	repo.Created = time.Now().UTC().Unix()
	repo.Updated = time.Now().UTC().Unix()

	return db.Update(func(t *bolt.Tx) error {
		userkey := []byte(user.Login)
		err := push(t, bucketUserRepos, userkey, repokey)
		if err != nil {
			return err
		}
		// err = push(t, bucketRepoUsers, repokey, userkey)
		// if err != nil {
		// 	return err
		// }
		return insert(t, bucketRepo, repokey, repo)
	})
}

// UpsertRepoParams inserts or updates the private
// environment parameters for the named repository.
func (db *DB) UpsertRepoParams(repo string, params map[string]string) error {
	key := []byte(repo)

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketRepoParams, key, params)
	})
}

// UpsertRepoKeys inserts or updates the private and
// public keypair for the named repository.
func (db *DB) UpsertRepoKeys(repo string, keypair *common.Keypair) error {
	key := []byte(repo)

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketRepoKeys, key, keypair)
	})
}

// DeleteRepo deletes the repository.
func (db *DB) DeleteRepo(repo *common.Repo) error {
	//TODO(benschumacher) rework this to use BoltDB's txn wrapper

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
func (db *DB) GetSubscriber(login, repo string) (*common.Subscriber, error) {
	sub := &common.Subscriber{}
	err := db.View(func(t *bolt.Tx) error {
		repokey := []byte(repo)

		// get the index of user tokens and unmarshal
		// to a string array.
		var keys [][]byte
		err := get(t, bucketUserRepos, []byte(login), &keys)
		if err != nil && err != ErrKeyNotFound {
			return err
		}

		for _, key := range keys {
			if bytes.Equal(repokey, key) {
				sub.Subscribed = true
				return nil
			}
		}
		return nil
	})
	return sub, err
}

// InsertSubscriber inserts a subscriber for the named
// repository.
func (db *DB) InsertSubscriber(login, repo string) error {
	return db.Update(func(t *bolt.Tx) error {
		userkey := []byte(login)
		repokey := []byte(repo)
		return push(t, bucketUserRepos, userkey, repokey)
	})
}

// DeleteSubscriber removes the subscriber by login for the
// named repository.
func (db *DB) DeleteSubscriber(login, repo string) error {
	return db.Update(func(t *bolt.Tx) error {
		userkey := []byte(login)
		repokey := []byte(repo)
		return splice(t, bucketUserRepos, userkey, repokey)
	})
}
