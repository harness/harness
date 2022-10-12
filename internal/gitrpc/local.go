// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type localStore struct {
	mutex sync.RWMutex
	files map[string]bool
}

func newLocalStore() *localStore {
	return &localStore{
		files: make(map[string]bool),
	}
}

func (store *localStore) Save(filePath string, data bytes.Buffer) (string, error) {
	err := os.MkdirAll(filepath.Dir(filePath), 0o777)
	if err != nil {
		return "", err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot create file: %w", err)
	}
	defer file.Close()

	_, err = data.WriteTo(file)
	if err != nil {
		return "", fmt.Errorf("cannot write to file: %w", err)
	}

	store.mutex.Lock()
	defer store.mutex.Unlock()

	store.files[filePath] = true

	return filePath, nil
}
