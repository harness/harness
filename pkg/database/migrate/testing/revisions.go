package migrate

import (
	. "github.com/drone/drone/pkg/database/migrate"
)

type Sample struct {
	ID   int64  `meddler:"id,pk"`
	Imel string `meddler:"imel"`
	Name string `meddler:"name"`
}

type RenameSample struct {
	ID    int64  `meddler:"id,pk"`
	Email string `meddler:"email"`
	Name  string `meddler:"name"`
}

type AddColumnSample struct {
	ID   int64  `meddler:"id,pk"`
	Imel string `meddler:"imel"`
	Name string `meddler:"name"`
	Url  string `meddler:"url"`
	Num  int64  `meddler:"num"`
}

// ---------- revision 1

type revision1 struct{}

func (r *revision1) Up(mg *MigrationDriver) error {
	_, err := mg.CreateTable("samples", []string{
		mg.T.Integer("id", PRIMARYKEY, AUTOINCREMENT),
		mg.T.String("imel", UNIQUE),
		mg.T.String("name"),
	})
	return err
}

func (r *revision1) Down(mg *MigrationDriver) error {
	_, err := mg.DropTable("samples")
	return err
}

func (r *revision1) Revision() int64 {
	return 1
}

// ---------- end of revision 1

// ---------- revision 2

type revision2 struct{}

func (r *revision2) Up(mg *MigrationDriver) error {
	_, err := mg.RenameTable("samples", "examples")
	return err
}

func (r *revision2) Down(mg *MigrationDriver) error {
	_, err := mg.RenameTable("examples", "samples")
	return err
}

func (r *revision2) Revision() int64 {
	return 2
}

// ---------- end of revision 2

// ---------- revision 3

type revision3 struct{}

func (r *revision3) Up(mg *MigrationDriver) error {
	if _, err := mg.AddColumn("samples", "url VARCHAR(255)"); err != nil {
		return err
	}
	_, err := mg.AddColumn("samples", "num INTEGER")
	return err
}

func (r *revision3) Down(mg *MigrationDriver) error {
	_, err := mg.DropColumns("samples", "num", "url")
	return err
}

func (r *revision3) Revision() int64 {
	return 3
}

// ---------- end of revision 3

// ---------- revision 4

type revision4 struct{}

func (r *revision4) Up(mg *MigrationDriver) error {
	_, err := mg.RenameColumns("samples", map[string]string{
		"imel": "email",
	})
	return err
}

func (r *revision4) Down(mg *MigrationDriver) error {
	_, err := mg.RenameColumns("samples", map[string]string{
		"email": "imel",
	})
	return err
}

func (r *revision4) Revision() int64 {
	return 4
}

// ---------- end of revision 4

// ---------- revision 5

type revision5 struct{}

func (r *revision5) Up(mg *MigrationDriver) error {
	_, err := mg.AddIndex("samples", []string{"url", "name"})
	return err
}

func (r *revision5) Down(mg *MigrationDriver) error {
	_, err := mg.DropIndex("samples", []string{"url", "name"})
	return err
}

func (r *revision5) Revision() int64 {
	return 5
}

// ---------- end of revision 5

// ---------- revision 6
type revision6 struct{}

func (r *revision6) Up(mg *MigrationDriver) error {
	_, err := mg.RenameColumns("samples", map[string]string{
		"url": "host",
	})
	return err
}

func (r *revision6) Down(mg *MigrationDriver) error {
	_, err := mg.RenameColumns("samples", map[string]string{
		"host": "url",
	})
	return err
}

func (r *revision6) Revision() int64 {
	return 6
}

// ---------- end of revision 6

// ---------- revision 7
type revision7 struct{}

func (r *revision7) Up(mg *MigrationDriver) error {
	_, err := mg.DropColumns("samples", "host", "num")
	return err
}

func (r *revision7) Down(mg *MigrationDriver) error {
	if _, err := mg.AddColumn("samples", "host VARCHAR(255)"); err != nil {
		return err
	}
	_, err := mg.AddColumn("samples", "num INSTEGER")
	return err
}

func (r *revision7) Revision() int64 {
	return 7
}

// ---------- end of revision 7

// ---------- revision 8
type revision8 struct{}

func (r *revision8) Up(mg *MigrationDriver) error {
	if _, err := mg.AddColumn("samples", "repo_id INTEGER"); err != nil {
		return err
	}
	_, err := mg.AddColumn("samples", "repo VARCHAR(255)")
	return err
}

func (r *revision8) Down(mg *MigrationDriver) error {
	_, err := mg.DropColumns("samples", "repo", "repo_id")
	return err
}

func (r *revision8) Revision() int64 {
	return 8
}

// ---------- end of revision 8

// ---------- revision 9
type revision9 struct{}

func (r *revision9) Up(mg *MigrationDriver) error {
	_, err := mg.RenameColumns("samples", map[string]string{
		"repo": "repository",
	})
	return err
}

func (r *revision9) Down(mg *MigrationDriver) error {
	_, err := mg.RenameColumns("samples", map[string]string{
		"repository": "repo",
	})
	return err
}

func (r *revision9) Revision() int64 {
	return 9
}

// ---------- end of revision 9

// ---------- revision 10

type revision10 struct{}

func (r *revision10) Revision() int64 {
	return 10
}

func (r *revision10) Up(mg *MigrationDriver) error {
	_, err := mg.ChangeColumn("samples", "email", "varchar(512) UNIQUE")
	return err
}

func (r *revision10) Down(mg *MigrationDriver) error {
	_, err := mg.ChangeColumn("samples", "email", "varchar(255) unique")
	return err
}

// ---------- end of revision 10
