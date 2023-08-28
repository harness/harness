// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.
package types

// Plugin represents a Harness plugin. It has an associated template stored
// in the spec field. The spec is used by the UI to provide a smart visual
// editor for adding plugins to YAML schema.
type Plugin struct {
	UID         string `db:"plugin_uid"              json:"uid"`
	Description string `db:"plugin_description"       json:"description"`
	Logo        string `db:"plugin_logo"              json:"logo"`
	// Spec is a YAML template to be used for the plugin.
	Spec string `db:"plugin_spec"                     json:"spec"`
}
