// Copyright 2021 Harness, Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE.md file.

//go:build !proxy
// +build !proxy

// Package dist embeds the static web server content.
package web

import (
	"embed"
)

//go:embed dist/*
var UI embed.FS
