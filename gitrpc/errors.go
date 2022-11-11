// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import "errors"

var ErrNoParamsProvided = errors.New("no params provided")
var ErrAlreadyExists = errors.New("already exists")
var ErrInvalidArgument = errors.New("invalid argument")
var ErrNotFound = errors.New("not found")
