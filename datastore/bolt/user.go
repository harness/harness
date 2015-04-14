package bolt

import (
	"time"

	"github.com/boltdb/bolt"
	"github.com/drone/drone/common"
)

// GetUser gets a user by user login.
func (db *DB) GetUser(login string) (*common.User, error) {
	user := &common.User{}
	key := []byte(login)

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketUser, key, user)
	})

	return user, err
}

// GetUserTokens gets a list of all tokens for
// the given user login.
func (db *DB) GetUserTokens(login string) ([]*common.Token, error) {
	t, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer t.Rollback()
	tokens := []*common.Token{}

	// get the index of user tokens and unmarshal
	// to a string array.
	var keys [][]byte
	err = get(t, bucketUserTokens, []byte(login), &keys)
	if err != nil && err != ErrKeyNotFound {
		return nil, err
	}

	// for each item in the index, get the repository
	// and append to the array
	for _, key := range keys {
		token := &common.Token{}
		raw := t.Bucket(bucketTokens).Get(key)
		err = decode(raw, token)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// GetUserRepos gets a list of repositories for the
// given user account.
func (db *DB) GetUserRepos(login string) ([]*common.Repo, error) {
	t, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer t.Rollback()
	repos := []*common.Repo{}
	user := &common.User{}

	// get the user struct from the database
	key := []byte(login)
	raw := t.Bucket(bucketUser).Get(key)
	err = decode(raw, user)
	if err != nil {
		return nil, err
	}

	// for each item in the index, get the repository
	// and append to the array
	for repoName, _ := range user.Repos {
		repo := &common.Repo{}
		key = []byte(repoName)
		raw = t.Bucket(bucketRepo).Get(key)
		if raw == nil {
			// this will happen when the repository has been deleted
			// TODO we should probably upate the index in this case.
			continue
		}
		err = decode(raw, repo)
		if err != nil {
			break
		}
		repos = append(repos, repo)
	}
	return repos, err
}

// GetUserCount gets a count of all registered users
// in the system.
func (db *DB) GetUserCount() (int, error) {
	var out int
	var err = db.View(func(t *bolt.Tx) error {
		out = t.Bucket(bucketUser).Stats().KeyN
		return nil
	})
	return out, err
}

// GetUserList gets a list of all registered users.
func (db *DB) GetUserList() ([]*common.User, error) {
	users := []*common.User{}
	err := db.View(func(t *bolt.Tx) error {
		return t.Bucket(bucketUser).ForEach(func(key, raw []byte) error {
			user := &common.User{}
			err := decode(raw, user)
			if err != nil {
				return err
			}
			users = append(users, user)
			return nil
		})
	})
	return users, err
}

// UpdateUser updates an existing user. If the user
// does not exists an error is returned.
func (db *DB) UpdateUser(user *common.User) error {
	key := []byte(user.Login)
	user.Updated = time.Now().UTC().Unix()

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketUser, key, user)
	})
}

// InsertUser inserts a new user into the datastore. If
// the user login already exists an error is returned.
func (db *DB) InsertUser(user *common.User) error {
	key := []byte(user.Login)
	user.Created = time.Now().UTC().Unix()
	user.Updated = time.Now().UTC().Unix()

	return db.Update(func(t *bolt.Tx) error {
		return insert(t, bucketUser, key, user)
	})
}

// DeleteUser deletes the user.
func (db *DB) DeleteUser(user *common.User) error {
	key := []byte(user.Login)
	// TODO(bradrydzewski) delete user subscriptions
	// TODO(bradrydzewski) delete user tokens

	return db.Update(func(t *bolt.Tx) error {
		return delete(t, bucketUser, key)
	})
}
