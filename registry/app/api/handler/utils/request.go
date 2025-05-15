//  Copyright 2023 Harness, Inc.
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

package utils

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

func GetFileReader(r *http.Request, formKey string) (*multipart.Part, string, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, "", err
	}

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, "", err
		}

		if part.FormName() == formKey {
			filename := part.FileName()
			return part, filename, nil
		}
	}

	return nil, "", fmt.Errorf("file not found in request")
}

// GetMultipartValues processes a multipart form and extracts both file parts and form values.
// It returns the file reader (if fileKey is found), filename, and a map of other form values.
// This function streams the multipart form without loading the entire content into memory.
func GetMultipartValues(r *http.Request, fileKey string,
	formKeys []string) (*multipart.Part, map[string]string, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, nil, err
	}

	formValues := make(map[string]string)
	var filePart *multipart.Part

	// Track which keys we still need to find
	keysToFind := make(map[string]bool)
	for _, key := range formKeys {
		keysToFind[key] = true
	}
	fileKeyNeeded := fileKey != ""

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		formName := part.FormName()

		// Check if this is the file part we're looking for
		if fileKeyNeeded && formName == fileKey {
			filePart = part
			fileKeyNeeded = false

			// If we've found all keys, we can stop
			if len(keysToFind) == 0 {
				break
			}
			continue
		}

		// Check if this is one of the form values we're looking for
		if _, ok := keysToFind[formName]; ok { //nolint:nestif
			// Read the form value (these are typically small)
			var valueBytes []byte
			buffer := make([]byte, 1024)

			for {
				n, err := part.Read(buffer)
				if n > 0 {
					valueBytes = append(valueBytes, buffer[:n]...)
				}
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return nil, nil, fmt.Errorf("error reading form value: %w", err)
				}
			}

			formValues[formName] = string(valueBytes)
			delete(keysToFind, formName)
			part.Close()

			// If we've found all form keys and the file (if needed), we can stop
			if len(keysToFind) == 0 && !fileKeyNeeded {
				break
			}
		} else {
			// Not a key we're looking for
			part.Close()
		}
	}

	// If fileKey was provided but not found, return an error
	if fileKey != "" && filePart == nil {
		return nil, formValues, fmt.Errorf("file part with key '%s' not found", fileKey)
	}

	return filePart, formValues, nil
}
