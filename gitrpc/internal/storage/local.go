// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStore struct{}

func NewLocalStore() *LocalStore {
	return &LocalStore{}
}

func (store *LocalStore) Save(filePath string, data io.Reader) (string, error) {
	err := os.MkdirAll(filepath.Dir(filePath), 0o777)
	if err != nil {
		return "", err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	if err != nil {
		return "", fmt.Errorf("cannot write to file: %w", err)
	}

	return filePath, nil
}
