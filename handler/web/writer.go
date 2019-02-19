// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
)

// indent the json-encoded API responses
var indent bool

func init() {
	indent, _ = strconv.ParseBool(
		os.Getenv("HTTP_JSON_INDENT"),
	)
}

var (
	// errInvalidToken is returned when the api request token is invalid.
	errInvalidToken = errors.New("Invalid or missing token")

	// errUnauthorized is returned when the user is not authorized.
	errUnauthorized = errors.New("Unauthorized")

	// errForbidden is returned when user access is forbidden.
	errForbidden = errors.New("Forbidden")

	// errNotFound is returned when a resource is not found.
	errNotFound = errors.New("Not Found")
)

// Error represents a json-encoded API error.
type Error struct {
	Message string `json:"message"`
}

// writeErrorCode writes the json-encoded error message to the response.
func writeErrorCode(w http.ResponseWriter, err error, status int) {
	writeJSON(w, &Error{Message: err.Error()}, status)
}

// writeError writes the json-encoded error message to the response
// with a 500 internal server error.
func writeError(w http.ResponseWriter, err error) {
	writeErrorCode(w, err, 500)
}

// writeNotFound writes the json-encoded error message to the response
// with a 404 not found status code.
func writeNotFound(w http.ResponseWriter, err error) {
	writeErrorCode(w, err, 404)
}

// writeUnauthorized writes the json-encoded error message to the response
// with a 401 unauthorized status code.
func writeUnauthorized(w http.ResponseWriter, err error) {
	writeErrorCode(w, err, 401)
}

// writeForbidden writes the json-encoded error message to the response
// with a 403 forbidden status code.
func writeForbidden(w http.ResponseWriter, err error) {
	writeErrorCode(w, err, 403)
}

// writeBadRequest writes the json-encoded error message to the response
// with a 400 bad request status code.
func writeBadRequest(w http.ResponseWriter, err error) {
	writeErrorCode(w, err, 400)
}

// writeJSON writes the json-encoded error message to the response
// with a 400 bad request status code.
func writeJSON(w http.ResponseWriter, v interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	if indent {
		enc.SetIndent("", "  ")
	}
	enc.Encode(v)
}
