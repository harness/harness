// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package types

// Plugin represents a Harness plugin. It has an associated template stored
// in the spec field. The spec is used by the UI to provide a smart visual
// editor for adding plugins to YAML schema.
type Plugin struct {
	UID         string `db:"plugin_uid"               json:"uid"`
	Description string `db:"plugin_description"       json:"description"`
	// Currently we only support step level plugins but more can be added in the future.
	Type    string `db:"plugin_type"              json:"type"`
	Version string `db:"plugin_version"           json:"version"`
	Logo    string `db:"plugin_logo"              json:"logo"`
	// Spec is a YAML template to be used for the plugin.
	Spec string `db:"plugin_spec"                     json:"spec"`
}

// Matches checks whether two plugins are identical.
// We can use reflection here, this is just easier to add on to
// when needed.
func (plugin *Plugin) Matches(v *Plugin) bool {
	if plugin.UID != v.UID {
		return false
	}
	if plugin.Description != v.Description {
		return false
	}
	if plugin.Spec != v.Spec {
		return false
	}
	if plugin.Version != v.Version {
		return false
	}
	if plugin.Logo != v.Logo {
		return false
	}
	return true
}
