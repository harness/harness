// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var dirs = []string{
	"postgres",
	"sqlite",
}

// TestMigrationFilesExtension checks if all files in the postgres and sqlite
// migration directories have the .sql extension.
func TestMigrationFilesExtension(t *testing.T) {
	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory %s: %v", dir, err)
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			fileName := file.Name()
			ext := filepath.Ext(fileName)
			assert.Equal(t, ".sql", ext, "File %s in %s directory should have .sql extension", fileName, dir)
		}
	}
}

// TestMigrationFilesNumbering checks if migration files are numbered correctly
// and each version has only 2 files (up and down).
func TestMigrationFilesNumbering(t *testing.T) {
	// Known issues with file numbering or files per version
	knownIssues := map[string]map[string]bool{
		"postgres": {
			"0000": true,
			"0001": true,
			"0002": true,
			"0003": true,
			"0004": true,
			"0005": true,
			"0006": true,
			"0007": true,
			"0008": true,
			"0009": true,
			"0010": true,
			"0011": true,
			"0012": true,
			"0021": true,
			"0026": true,
			"0029": true,
			"0033": true,
			"0058": true,
			"0059": true,
			"0106": true,
			"0111": true,
			"0115": true,
			"0122": true,
			"0134": true,
			"0155": true,
		},
		"sqlite": {
			"0000": true,
			"0001": true,
			"0002": true,
			"0003": true,
			"0004": true,
			"0005": true,
			"0006": true,
			"0007": true,
			"0008": true,
			"0009": true,
			"0010": true,
			"0011": true,
			"0012": true,
			"0021": true,
			"0029": true,
			"0033": true,
			"0058": true,
			"0059": true,
			"0097": true,
			"0106": true,
			"0111": true,
			"0115": true,
			"0122": true,
			"0134": true,
			"0135": true,
			"0155": true,
		},
	}

	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("Failed to read directory %s: %v", dir, err)
		}

		// Map to count files per version
		versionCount := make(map[string][]string)
		allVersions := make(map[int]bool)

		// Check if files are numbered correctly
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			fileName := file.Name()

			// Extract version number
			parts := strings.Split(fileName, "_")
			if len(parts) < 2 {
				t.Fatalf("File %s in %s directory has invalid naming format", fileName, dir)
			}

			versionStr := parts[0]

			versionInt, err := strconv.Atoi(versionStr)
			if err != nil {
				t.Errorf("File %s in %s directory has invalid version number: %v", fileName, dir, err)
				continue
			}
			allVersions[versionInt] = true

			if _, ok := knownIssues[dir][versionStr]; ok {
				continue
			}

			versionCount[versionStr] = append(versionCount[versionStr], fileName)
		}

		// Check if each version has exactly 2 files (up and down)
		for version, f := range versionCount {
			if len(f) != 2 {
				t.Errorf("Version %s in %s directory has %d files, expected 2 files (up and down)", version, dir,
					len(f))
				continue
			}
			name1, ext1, err := getFileParts(f[0])
			if err != nil {
				t.Errorf("File %s in %s directory has invalid name format: %v", f[0], dir, err)
			}
			name2, ext2, err := getFileParts(f[1])
			if err != nil {
				t.Errorf("File %s in %s directory has invalid name format: %v", f[0], dir, err)
			}

			if name1 != name2 {
				t.Errorf("Name mismatch for version %s in %s directory: %s != %s", version, dir, files[0], files[1])
			}

			if !((ext1 == "up.sql" && ext2 == "down.sql") || (ext2 == "up.sql" && ext1 == "down.sql")) { //nolint:staticcheck
				t.Errorf("Extension mismatch for version %s in %s directory: %s != %s", version, dir, files[0],
					files[1])
			}
		}

		// Check if all versions are in sequence order
		maxVersion := -1
		for versionInt := range allVersions {
			if versionInt > maxVersion {
				maxVersion = versionInt
			}
		}
		if len(allVersions)-1 != maxVersion {
			t.Errorf("Number of versions mismatch for %s in %s directory: %d != %d", dir, dir, len(allVersions),
				maxVersion-1)
		}
	}
}

func getFileParts(filename string) (name, ext string, err error) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid filename %s", filename)
	}
	parts2 := strings.SplitN(parts[1], ".", 2)
	if len(parts2) != 2 {
		return "", "", fmt.Errorf("invalid versioning %s", filename)
	}
	return parts2[0], parts2[1], nil
}

// TestMigrationFilesExistInBothDatabases checks that the same migration files
// exist in both postgres and sqlite directories, with exceptions for known differences.
func TestMigrationFilesExistInBothDatabases(t *testing.T) {
	// Known files that only exist in one database but not the other
	knownExceptions := map[string]bool{
		// Files only in postgres
		"0000_create_extension_btree.up.sql":              true,
		"0000_create_extension_citext.up.sql":             true,
		"0000_create_extension_trgm.up.sql":               true,
		"0026_alter_repo_drop_job_id.up.sql":              true, // Different naming in sqlite
		"0026_alter_repo_drop_join_id.down.sql":           true,
		"0097_update_ar_generic_artifact_tables.down.sql": true, // Different naming in sqlite
		"0097_update_ar_generic_artifact_tables.up.sql":   true, // Different naming in sqlite

		// Files only in sqlite
		"0097_update_ar_generic_artiface_tables.up.sql": true,
		"0026_alter_repo_drop_job_id.down.sql":          true, // Different naming in postgres
	}

	// Read postgres directory
	postgresFiles, err := os.ReadDir("postgres")
	if err != nil {
		t.Fatalf("Failed to read postgres directory: %v", err)
	}

	// Read sqlite directory
	sqliteFiles, err := os.ReadDir("sqlite")
	if err != nil {
		t.Fatalf("Failed to read sqlite directory: %v", err)
	}

	// Create maps of file names for easier comparison
	postgresFileMap := make(map[string]bool)
	sqliteFileMap := make(map[string]bool)

	for _, file := range postgresFiles {
		if !file.IsDir() {
			postgresFileMap[file.Name()] = true
		}
	}

	for _, file := range sqliteFiles {
		if !file.IsDir() {
			sqliteFileMap[file.Name()] = true
		}
	}

	// Check postgres files exist in sqlite
	for fileName := range postgresFileMap {
		if _, isException := knownExceptions[fileName]; isException {
			continue // Skip known exceptions
		}

		if !sqliteFileMap[fileName] {
			t.Errorf("File %s exists in postgres but not in sqlite", fileName)
		}
	}

	// Check sqlite files exist in postgres
	for fileName := range sqliteFileMap {
		if _, isException := knownExceptions[fileName]; isException {
			continue // Skip known exceptions
		}

		if !postgresFileMap[fileName] {
			t.Errorf("File %s exists in sqlite but not in postgres", fileName)
		}
	}
}
