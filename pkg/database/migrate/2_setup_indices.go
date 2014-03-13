package migrate

type rev2nd struct{}

var SetupIndices = &rev2nd{}

func (r *rev2nd) Revision() int64 {
	return 2
}

func (r *rev2nd) Up(mg *MigrationDriver) error {
	if _, err := mg.AddIndex("members", []string{"team_id", "user_id"}, "unique"); err != nil {
		return err
	}

	if _, err := mg.AddIndex("members", []string{"team_id"}); err != nil {
		return err
	}

	if _, err := mg.AddIndex("members", []string{"user_id"}); err != nil {
		return err
	}

	if _, err := mg.AddIndex("commits", []string{"repo_id", "hash", "branch"}, "unique"); err != nil {
		return err
	}

	if _, err := mg.AddIndex("commits", []string{"repo_id"}); err != nil {
		return err
	}

	if _, err := mg.AddIndex("commits", []string{"repo_id", "branch"}); err != nil {
		return err
	}

	if _, err := mg.AddIndex("repos", []string{"team_id"}); err != nil {
		return err
	}

	if _, err := mg.AddIndex("repos", []string{"user_id"}); err != nil {
		return err
	}

	if _, err := mg.AddIndex("builds", []string{"commit_id"}); err != nil {
		return err
	}

	_, err := mg.AddIndex("builds", []string{"commit_id", "slug"})

	return err
}

func (r *rev2nd) Down(mg *MigrationDriver) error {
	if _, err := mg.DropIndex("builds", []string{"commit_id", "slug"}); err != nil {
		return err
	}
	if _, err := mg.DropIndex("builds", []string{"commit_id"}); err != nil {
		return err
	}
	if _, err := mg.DropIndex("repos", []string{"user_id"}); err != nil {
		return err
	}
	if _, err := mg.DropIndex("repos", []string{"team_id"}); err != nil {
		return err
	}
	if _, err := mg.DropIndex("commits", []string{"repo_id", "branch"}); err != nil {
		return err
	}
	if _, err := mg.DropIndex("commits", []string{"repo_id"}); err != nil {
		return err
	}
	if _, err := mg.DropIndex("commits", []string{"repo_id", "hash", "branch"}); err != nil {
		return err
	}
	if _, err := mg.DropIndex("members", []string{"user_id"}); err != nil {
		return err
	}
	if _, err := mg.DropIndex("members", []string{"team_id"}); err != nil {
		return err
	}
	_, err := mg.DropIndex("members", []string{"team_id", "user_id"})
	return err
}
