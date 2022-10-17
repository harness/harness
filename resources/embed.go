// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package resources

import (
	"embed"
	"fmt"
	"strings"
)

var (
	//go:embed gitignore
	gitignore embed.FS

	//go:embed license
	licence embed.FS
)

// Licenses returns map of licences in license folder.
func Licenses() ([]byte, error) {
	return licence.ReadFile("license/index.json")
}

// ReadLicense reads licence from license folder.
func ReadLicense(name string) ([]byte, error) {
	content, err := licence.ReadFile(fmt.Sprintf("license/%s.txt", name))
	if err != nil {
		return nil, err
	}
	return content, err
}

// GitIgnores lists all files in gitignore folder and return file names.
func GitIgnores() ([]string, error) {
	entries, err := gitignore.ReadDir("gitignore")
	files := make([]string, len(entries))
	if err != nil {
		return []string{}, err
	}
	for i, filename := range entries {
		files[i] = strings.ReplaceAll(filename.Name(), ".gitignore", "")
	}
	return files, nil
}

// ReadGitIgnore reads gitignore file from license folder.
func ReadGitIgnore(name string) ([]byte, error) {
	return gitignore.ReadFile(fmt.Sprintf("gitignore/%s.gitignore", name))
}
