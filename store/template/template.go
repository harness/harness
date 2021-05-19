// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package template

import (
	"context"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

// New returns a new Template database store.
func New(db *db.DB) core.TemplateStore {
	return &templateStore{
		db: db,
	}
}

type templateStore struct {
	db *db.DB
}

func (s *templateStore) List(ctx context.Context, id int64) ([]*core.Template, error) {
	var out []*core.Template
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{"template_id": id}
		stmt, args, err := binder.BindNamed(queryRepo, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(stmt, args...)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

func (s *templateStore) Find(ctx context.Context, id int64) (*core.Template, error) {
	out := &core.Template{Id: id}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params, err := toParams(out)
		if err != nil {
			return err
		}
		query, args, err := binder.BindNamed(queryKey, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

func (s *templateStore) FindName(ctx context.Context, id int64, name string) (*core.Template, error) {
	out := &core.Template{Name: name, Id: id}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params, err := toParams(out)
		if err != nil {
			return err
		}
		query, args, err := binder.BindNamed(queryName, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

func (s *templateStore) Create(ctx context.Context, template *core.Template) error {
	if s.db.Driver() == db.Postgres {
		return s.createPostgres(ctx, template)
	}
	return s.create(ctx, template)
}

func (s *templateStore) create(ctx context.Context, template *core.Template) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params, err := toParams(template)
		if err != nil {
			return err
		}
		stmt, args, err := binder.BindNamed(stmtInsert, params)
		if err != nil {
			return err
		}
		res, err := execer.Exec(stmt, args...)
		if err != nil {
			return err
		}
		template.Id, err = res.LastInsertId()
		return err
	})
}

func (s *templateStore) createPostgres(ctx context.Context, template *core.Template) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params, err := toParams(template)
		if err != nil {
			return err
		}
		stmt, args, err := binder.BindNamed(stmtInsertPostgres, params)
		if err != nil {
			return err
		}
		return execer.QueryRow(stmt, args...).Scan(&template.Id)
	})
}

func (s *templateStore) Update(ctx context.Context, template *core.Template) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params, err := toParams(template)
		if err != nil {
			return err
		}
		stmt, args, err := binder.BindNamed(stmtUpdate, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

func (s *templateStore) Delete(ctx context.Context, template *core.Template) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params, err := toParams(template)
		if err != nil {
			return err
		}
		stmt, args, err := binder.BindNamed(stmtDelete, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

const queryKey = queryBase + `
FROM template
WHERE template_id = :template_id
LIMIT 1
`

const queryBase = `
SELECT
 template_id
,template_name
,template_data
,template_created
,template_updated
`

const queryRepo = queryBase + `
FROM template
WHERE template_id = :template_id
ORDER BY template_name
`
const stmtInsert = `
INSERT INTO template (
 template_id
,template_name
,template_data
,template_created
,template_updated
) VALUES (
 :template_id
,:template_name
,:template_data
,:template_created
,:template_updated
)
`

const stmtUpdate = `
UPDATE template SET
template_name = :template_name
,template_data = :template_data
,template_updated = :template_updated
WHERE template_id = :template_id
`

const stmtDelete = `
DELETE FROM template
WHERE template_id = :template_id
`
const queryName = queryBase + `
FROM template
WHERE template_name = :template_name
  AND template_id = :template_id
LIMIT 1
`

const stmtInsertPostgres = stmtInsert + `
RETURNING template_id
`
