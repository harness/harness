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

package blob

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewFileSystemStore(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "basic config",
			config: Config{
				Bucket: "/tmp/test-storage",
			},
			expected: "/tmp/test-storage",
		},
		{
			name: "empty bucket",
			config: Config{
				Bucket: "",
			},
			expected: "",
		},
		{
			name: "relative path",
			config: Config{
				Bucket: "relative/path",
			},
			expected: "relative/path",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, err := NewFileSystemStore(test.config)
			if err != nil {
				t.Fatalf("unexpected error creating filesystem store: %v", err)
			}

			fsStore, ok := store.(*FileSystemStore)
			if !ok {
				t.Fatal("expected FileSystemStore type")
			}

			if fsStore.basePath != test.expected {
				t.Errorf("expected base path %q, got %q", test.expected, fsStore.basePath)
			}
		})
	}
}

func TestFileSystemStore_Upload(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "blob-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := &FileSystemStore{basePath: tempDir}
	ctx := context.Background()

	tests := []struct {
		name        string
		filePath    string
		content     string
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name:        "simple file upload",
			filePath:    "test.txt",
			content:     "hello world",
			expectError: false,
		},
		{
			name:        "nested directory upload",
			filePath:    "subdir/nested/file.txt",
			content:     "nested content",
			expectError: false,
		},
		{
			name:        "empty file",
			filePath:    "empty.txt",
			content:     "",
			expectError: false,
		},
		{
			name:        "file with special characters",
			filePath:    "special-file_123.txt",
			content:     "special content",
			expectError: false,
		},
		{
			name:        "large content",
			filePath:    "large.txt",
			content:     strings.Repeat("a", 10000),
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := strings.NewReader(test.content)
			err := store.Upload(ctx, reader, test.filePath)

			if test.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if test.errorCheck != nil && !test.errorCheck(err) {
					t.Errorf("error check failed for error: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify file was created and has correct content
			fullPath := filepath.Join(tempDir, test.filePath)
			data, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatalf("failed to read uploaded file: %v", err)
			}

			if string(data) != test.content {
				t.Errorf("expected content %q, got %q", test.content, string(data))
			}
		})
	}
}

func TestFileSystemStore_Upload_DirectoryCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blob-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := &FileSystemStore{basePath: tempDir}
	ctx := context.Background()

	// Test that nested directories are created automatically
	filePath := "level1/level2/level3/file.txt"
	content := "nested file content"
	reader := strings.NewReader(content)

	err = store.Upload(ctx, reader, filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the directory structure was created
	fullPath := filepath.Join(tempDir, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("file was not created")
	}

	// Verify directory exists
	dirPath := filepath.Dir(fullPath)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Error("directory was not created")
	}
}

func TestFileSystemStore_Download(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blob-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := &FileSystemStore{basePath: tempDir}
	ctx := context.Background()

	// Create test files
	testFiles := map[string]string{
		"test1.txt":        "content1",
		"subdir/test2.txt": "content2",
		"empty.txt":        "",
		"large.txt":        strings.Repeat("x", 5000),
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name        string
		filePath    string
		expected    string
		expectError bool
		errorType   error
	}{
		{
			name:        "existing file",
			filePath:    "test1.txt",
			expected:    "content1",
			expectError: false,
		},
		{
			name:        "nested file",
			filePath:    "subdir/test2.txt",
			expected:    "content2",
			expectError: false,
		},
		{
			name:        "empty file",
			filePath:    "empty.txt",
			expected:    "",
			expectError: false,
		},
		{
			name:        "large file",
			filePath:    "large.txt",
			expected:    strings.Repeat("x", 5000),
			expectError: false,
		},
		{
			name:        "non-existent file",
			filePath:    "nonexistent.txt",
			expectError: true,
			errorType:   ErrNotFound,
		},
		{
			name:        "non-existent nested file",
			filePath:    "nonexistent/file.txt",
			expectError: true,
			errorType:   ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader, err := store.Download(ctx, test.filePath)

			if test.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if test.errorType != nil && !errors.Is(err, test.errorType) {
					t.Errorf("expected error type %v, got %v", test.errorType, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			defer reader.Close()

			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("failed to read downloaded content: %v", err)
			}

			if string(data) != test.expected {
				t.Errorf("expected content %q, got %q", test.expected, string(data))
			}
		})
	}
}

func TestFileSystemStore_GetSignedURL(t *testing.T) {
	store := &FileSystemStore{basePath: "/tmp"}
	ctx := context.Background()

	// Test that GetSignedURL returns ErrNotSupported
	url, err := store.GetSignedURL(ctx, "test.txt", time.Now().Add(time.Hour))

	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}

	if url != "" {
		t.Errorf("expected empty URL, got %q", url)
	}
}

func TestFileSystemStore_GetSignedURL_WithOptions(t *testing.T) {
	store := &FileSystemStore{basePath: "/tmp"}
	ctx := context.Background()

	// Test with various options
	options := []SignURLOption{
		SignWithMethod("POST"),
		SignWithContentType("application/json"),
		SignWithHeaders([]string{"Authorization"}),
	}

	url, err := store.GetSignedURL(ctx, "test.txt", time.Now().Add(time.Hour), options...)

	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}

	if url != "" {
		t.Errorf("expected empty URL, got %q", url)
	}
}

func TestFileSystemStore_Upload_ErrorCases(t *testing.T) {
	// Test with invalid base path (read-only directory)
	if os.Getuid() != 0 { // Skip if running as root
		readOnlyDir, err := os.MkdirTemp("", "readonly-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(readOnlyDir)

		// Make directory read-only
		if err := os.Chmod(readOnlyDir, 0444); err != nil {
			t.Fatalf("failed to make directory read-only: %v", err)
		}

		store := &FileSystemStore{basePath: readOnlyDir}
		ctx := context.Background()

		reader := strings.NewReader("test content")
		err = store.Upload(ctx, reader, "test.txt")
		if err == nil {
			t.Error("expected error when writing to read-only directory")
		}
	}
}

func TestFileSystemStore_Interface(t *testing.T) {
	// Test that FileSystemStore implements Store interface
	var _ Store = &FileSystemStore{}

	// Test that NewFileSystemStore returns Store interface
	config := Config{Bucket: "/tmp"}
	store, err := NewFileSystemStore(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = store
}

func TestFileSystemStore_Move(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blob-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := &FileSystemStore{basePath: tempDir}
	ctx := context.Background()

	tests := []struct {
		name        string
		srcPath     string
		dstPath     string
		content     string
		expectError bool
		errorType   error
	}{
		{
			name:        "move file same directory",
			srcPath:     "src.txt",
			dstPath:     "dst.txt",
			content:     "test content",
			expectError: false,
		},
		{
			name:        "move file to nested directory",
			srcPath:     "src2.txt",
			dstPath:     "nested/dir/dst2.txt",
			content:     "nested content",
			expectError: false,
		},
		{
			name:        "move non-existent file",
			srcPath:     "nonexistent.txt",
			dstPath:     "dst.txt",
			content:     "",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup: create source file if content is provided
			if test.content != "" {
				srcFullPath := filepath.Join(tempDir, test.srcPath)
				if err := os.WriteFile(srcFullPath, []byte(test.content), 0600); err != nil {
					t.Fatalf("failed to create source file: %v", err)
				}
			}

			err := store.Move(ctx, test.srcPath, test.dstPath)

			if test.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify source file no longer exists
			srcFullPath := filepath.Join(tempDir, test.srcPath)
			if _, err := os.Stat(srcFullPath); !os.IsNotExist(err) {
				t.Error("source file should not exist after move")
			}

			// Verify destination file exists with correct content
			dstFullPath := filepath.Join(tempDir, test.dstPath)
			data, err := os.ReadFile(dstFullPath)
			if err != nil {
				t.Fatalf("failed to read destination file: %v", err)
			}

			if string(data) != test.content {
				t.Errorf("expected content %q, got %q", test.content, string(data))
			}
		})
	}
}

func TestFileSystemStore_Delete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blob-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := &FileSystemStore{basePath: tempDir}
	ctx := context.Background()

	tests := []struct {
		name        string
		filePath    string
		content     string
		expectError bool
		errorType   error
	}{
		{
			name:        "delete existing file",
			filePath:    "to-delete.txt",
			content:     "delete me",
			expectError: false,
		},
		{
			name:        "delete nested file",
			filePath:    "nested/to-delete.txt",
			content:     "delete nested",
			expectError: false,
		},
		{
			name:        "delete non-existent file",
			filePath:    "nonexistent.txt",
			content:     "",
			expectError: true,
			errorType:   ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup: create file if content is provided
			if test.content != "" {
				fullPath := filepath.Join(tempDir, test.filePath)
				dir := filepath.Dir(fullPath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(test.content), 0600); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			}

			err := store.Delete(ctx, test.filePath)

			if test.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if test.errorType != nil && !errors.Is(err, test.errorType) {
					t.Errorf("expected error type %v, got %v", test.errorType, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify file no longer exists
			fullPath := filepath.Join(tempDir, test.filePath)
			if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
				t.Error("file should not exist after delete")
			}
		})
	}
}
