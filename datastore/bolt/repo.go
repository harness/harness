package bolt

import (
	"bytes"
	"time"

	"github.com/boltdb/bolt"
	"github.com/drone/drone/common"
)

// Repo returns the repository with the given name.
func (db *DB) Repo(repo string) (*common.Repo, error) {
	repo_ := &common.Repo{}
	key := []byte(repo)

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketRepo, key, repo_)
	})

	return repo_, err
}

// RepoList returns a list of repositories for the
// given user account.
func (db *DB) RepoList(login string) ([]*common.Repo, error) {
	repos := []*common.Repo{}
	err := db.View(func(t *bolt.Tx) error {
		// get the index of user tokens and unmarshal
		// to a string array.
		var keys [][]byte
		err := get(t, bucketUserRepos, []byte(login), &keys)
		if err != nil && err != ErrKeyNotFound {
			return err
		}

		// for each item in the index, get the repository
		// and append to the array
		for _, key := range keys {
			repo := &common.Repo{}
			err := get(t, bucketRepo, key, repo)
			if err == ErrKeyNotFound {
				// TODO if we come across ErrKeyNotFound it means
				// we need to re-build the index
				continue
			} else if err != nil {
				return err
			}
			repos = append(repos, repo)
		}
		return nil
	})

	return repos, err
}

// RepoParams returns the private environment parameters
// for the given repository.
func (db *DB) RepoParams(repo string) (map[string]string, error) {
	params := map[string]string{}
	key := []byte(repo)

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketRepoParams, key, &params)
	})

	return params, err
}

// RepoKeypair returns the private and public rsa keys
// for the given repository.
func (db *DB) RepoKeypair(repo string) (*common.Keypair, error) {
	keypair := &common.Keypair{}
	key := []byte(repo)

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketRepoKeys, key, keypair)
	})

	return keypair, err
}

// SetRepo inserts or updates a repository.
func (db *DB) SetRepo(repo *common.Repo) error {
	key := []byte(repo.FullName)
	repo.Updated = time.Now().UTC().Unix()

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketRepo, key, repo)
	})
}

// SetRepoNotExists updates a repository. If the repository
// already exists ErrConflict is returned.
func (db *DB) SetRepoNotExists(user *common.User, repo *common.Repo) error {
	repokey := []byte(repo.FullName)
	repo.Created = time.Now().UTC().Unix()
	repo.Updated = time.Now().UTC().Unix()

	return db.Update(func(t *bolt.Tx) error {
		userkey := []byte(user.Login)
		err := push(t, bucketUserRepos, userkey, repokey)
		if err != nil {
			return err
		}
		return insert(t, bucketRepo, repokey, repo)
	})
}

// SetRepoParams inserts or updates the private
// environment parameters for the named repository.
func (db *DB) SetRepoParams(repo string, params map[string]string) error {
	key := []byte(repo)

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketRepoParams, key, params)
	})
}

// SetRepoKeypair inserts or updates the private and
// public keypair for the named repository.
func (db *DB) SetRepoKeypair(repo string, keypair *common.Keypair) error {
	key := []byte(repo)

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketRepoKeys, key, keypair)
	})
}

// DelRepo deletes the repository.
func (db *DB) DelRepo(repo *common.Repo) error {
	key := []byte(repo.FullName)

	return db.Update(func(t *bolt.Tx) error {
		err := t.Bucket(bucketRepo).Delete(key)
		if err != nil {
			return err
		}
		t.Bucket(bucketRepoKeys).Delete(key)
		t.Bucket(bucketRepoParams).Delete(key)
		t.Bucket(bucketBuildSeq).Delete(key)
		deleteWithPrefix(t, bucketBuild, append(key, '/'))
		deleteWithPrefix(t, bucketBuildLogs, append(key, '/'))
		deleteWithPrefix(t, bucketBuildStatus, append(key, '/'))

		return err
	})
}

// Subscribed returns true if the user is subscribed
// to the named repository.
//
// TODO (bradrydzewski) we are currently storing the subscription
// data in a wrapper element called common.Subscriber. This is
// no longer necessary.
func (db *DB) Subscribed(login, repo string) (bool, error) {
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
	return sub.Subscribed, err
}

// SetSubscriber inserts a subscriber for the named
// repository.
func (db *DB) SetSubscriber(login, repo string) error {
	return db.Update(func(t *bolt.Tx) error {
		userkey := []byte(login)
		repokey := []byte(repo)
		return push(t, bucketUserRepos, userkey, repokey)
	})
}

// DelSubscriber removes the subscriber by login for the
// named repository.
func (db *DB) DelSubscriber(login, repo string) error {
	return db.Update(func(t *bolt.Tx) error {
		userkey := []byte(login)
		repokey := []byte(repo)
		return splice(t, bucketUserRepos, userkey, repokey)
	})
}
