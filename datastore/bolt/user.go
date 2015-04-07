package bolt

import (
	"time"

	"github.com/drone/drone/common"
)

// GetUser gets a user by user login.
func (db *DB) GetUser(login string) (*common.User, error) {
	user := &common.User{}
	key := []byte(login)
	err := get(db, bucketUser, key, user)
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
	key := []byte(login)
	raw := t.Bucket(bucketUserTokens).Get(key)
	keys := [][]byte{}
	err = decode(raw, &keys)
	if err != nil {
		return tokens, err
	}

	// for each item in the index, get the repository
	// and append to the array
	for _, key := range keys {
		token := &common.Token{}
		raw = t.Bucket(bucketTokens).Get(key)
		err = decode(raw, token)
		if err != nil {
			break
		}
		tokens = append(tokens, token)
	}
	return tokens, err
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

	// get the index of user repos and unmarshal
	// to a string array.
	key := []byte(login)
	raw := t.Bucket(bucketUserRepos).Get(key)
	keys := [][]byte{}
	err = decode(raw, &keys)
	if err != nil {
		return repos, err
	}

	// for each item in the index, get the repository
	// and append to the array
	for _, key := range keys {
		repo := &common.Repo{}
		raw = t.Bucket(bucketRepo).Get(key)
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
	t, err := db.Begin(false)
	if err != nil {
		return 0, err
	}
	defer t.Rollback()
	return t.Bucket(bucketUser).Stats().KeyN, nil
}

// GetUserList gets a list of all registered users.
func (db *DB) GetUserList() ([]*common.User, error) {
	t, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer t.Rollback()
	users := []*common.User{}
	err = t.Bucket(bucketUser).ForEach(func(key, raw []byte) error {
		user := &common.User{}
		err := decode(raw, user)
		users = append(users, user)
		return err
	})
	return users, err
}

// UpdateUser updates an existing user. If the user
// does not exists an error is returned.
func (db *DB) UpdateUser(user *common.User) error {
	key := []byte(user.Login)
	user.Updated = time.Now().UTC().Unix()
	return update(db, bucketUser, key, user)
}

// InsertUser inserts a new user into the datastore. If
// the user login already exists an error is returned.
func (db *DB) InsertUser(user *common.User) error {
	key := []byte(user.Login)
	user.Created = time.Now().UTC().Unix()
	user.Updated = time.Now().UTC().Unix()
	return insert(db, bucketUser, key, user)
}

// DeleteUser deletes the user.
func (db *DB) DeleteUser(user *common.User) error {
	key := []byte(user.Login)
	// TODO(bradrydzewski) delete user subscriptions
	// TODO(bradrydzewski) delete user tokens
	return delete(db, bucketUser, key)
}
