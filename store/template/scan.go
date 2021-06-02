// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package template

import (
	"database/sql"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

// helper function converts the Template structure to a set
// of named query parameters.
func toParams(template *core.Template) (map[string]interface{}, error) {
	return map[string]interface{}{
		"template_id":        template.Id,
		"template_name":      template.Name,
		"template_namespace": template.Namespace,
		"template_data":      template.Data,
		"template_created":   template.Created,
		"template_updated":   template.Updated,
	}, nil
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRow(scanner db.Scanner, dst *core.Template) error {
	err := scanner.Scan(
		&dst.Id,
		&dst.Name,
		&dst.Namespace,
		&dst.Data,
		&dst.Created,
		&dst.Updated,
	)
	if err != nil {
		return err
	}
	return nil
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRows(rows *sql.Rows) ([]*core.Template, error) {
	defer rows.Close()

	template := []*core.Template{}
	for rows.Next() {
		tem := new(core.Template)
		err := scanRow(rows, tem)
		if err != nil {
			return nil, err
		}
		template = append(template, tem)
	}
	return template, nil
}
