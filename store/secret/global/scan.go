// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package global

import (
	"database/sql"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
	"github.com/drone/drone/store/shared/encrypt"
)

// helper function converts the User structure to a set
// of named query parameters.
func toParams(encrypt encrypt.Encrypter, secret *core.Secret) (map[string]interface{}, error) {
	ciphertext, err := encrypt.Encrypt(secret.Data)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"secret_id":                secret.ID,
		"secret_namespace":         secret.Namespace,
		"secret_name":              secret.Name,
		"secret_type":              secret.Type,
		"secret_data":              ciphertext,
		"secret_pull_request":      secret.PullRequest,
		"secret_pull_request_push": secret.PullRequestPush,
	}, nil
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRow(encrypt encrypt.Encrypter, scanner db.Scanner, dst *core.Secret) error {
	var ciphertext []byte
	err := scanner.Scan(
		&dst.ID,
		&dst.Namespace,
		&dst.Name,
		&dst.Type,
		&ciphertext,
		&dst.PullRequest,
		&dst.PullRequestPush,
	)
	if err != nil {
		return err
	}
	plaintext, err := encrypt.Decrypt(ciphertext)
	if err != nil {
		return err
	}
	dst.Data = plaintext
	return nil
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRows(encrypt encrypt.Encrypter, rows *sql.Rows) ([]*core.Secret, error) {
	defer rows.Close()

	secrets := []*core.Secret{}
	for rows.Next() {
		sec := new(core.Secret)
		err := scanRow(encrypt, rows, sec)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, sec)
	}
	return secrets, nil
}
