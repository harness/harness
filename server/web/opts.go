// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import "time"

// Options defines website handler options.
type Options struct {
	sync time.Duration
	path string
	docs string
}

// Option configures the website handler.
type Option func(*Options)

// WithSync configures the website handler with the duration value
// used to determine if the user account requires synchronization.
func WithSync(d time.Duration) Option {
	return func(o *Options) {
		o.sync = d
	}
}

// WithDir configures the website handler with the directory value
// used to serve the website from the local filesystem.
func WithDir(s string) Option {
	return func(o *Options) {
		o.path = s
	}
}

// WithDocs configures the website handler with the documentation
// website address, which should be included in the user interface.
func WithDocs(s string) Option {
	return func(o *Options) {
		o.docs = s
	}
}
