package fixtures

import (
	"database/sql"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

func LoadCommits(db *sql.DB) {
	meddler.Insert(db, "commits", &model.Commit{
		RepoID:      2,
		Status:      "Success",
		Started:     1398065345,
		Finished:    1398069999,
		Duration:    854,
		Sha:         "4e81eca185897c2d0cf81f5bc12623550c2ef952",
		Branch:      "dev",
		PullRequest: "3",
		Author:      "drcooper@caltech.edu",
		Gravatar:    "ab23a88a3ed77ecdfeb894c0eaf2817a",
		Timestamp:   "Wed Apr 23 01:00:00 2014 -0700",
		Message:     "a commit message",
		Created:     1398065343,
		Updated:     1398065344,
	})

	meddler.Insert(db, "commits", &model.Commit{
		RepoID:      2,
		Status:      "Success",
		Started:     1398065345,
		Finished:    1398069999,
		Duration:    854,
		Sha:         "4e81eca185897c2d0cf81f5bc12623550c2ef952",
		Branch:      "master",
		PullRequest: "4",
		Author:      "drcooper@caltech.edu",
		Gravatar:    "ab23a88a3ed77ecdfeb894c0eaf2817a",
		Timestamp:   "Wed Apr 23 01:01:00 2014 -0700",
		Message:     "a commit message",
		Created:     1398065343,
		Updated:     1398065344,
	})

	meddler.Insert(db, "commits", &model.Commit{
		RepoID:      2,
		Status:      "Success",
		Started:     1398065345,
		Finished:    1398069999,
		Duration:    854,
		Sha:         "7253f6545caed41fb8f5a6fcdb3abc0b81fa9dbf",
		Branch:      "master",
		PullRequest: "5",
		Author:      "drcooper@caltech.edu",
		Gravatar:    "ab23a88a3ed77ecdfeb894c0eaf2817a",
		Timestamp:   "Wed Apr 23 01:02:38 2014 -0700",
		Message:     "a commit message",
		Created:     1398065343,
		Updated:     1398065344,
	})

	meddler.Insert(db, "commits", &model.Commit{
		RepoID:    1,
		Status:    "Success",
		Started:   1398065345,
		Finished:  1398069999,
		Duration:  854,
		Sha:       "d12c9e5a11982f71796ad909c93551b16fba053e",
		Branch:    "dev",
		Author:    "drcooper@caltech.edu",
		Gravatar:  "ab23a88a3ed77ecdfeb894c0eaf2817a",
		Timestamp: "Wed Apr 23 02:00:00 2014 -0700",
		Message:   "a commit message",
		Created:   1398065343,
		Updated:   1398065344,
	})

	meddler.Insert(db, "commits", &model.Commit{
		RepoID:    1,
		Status:    "Started",
		Started:   1398065345,
		Finished:  0,
		Duration:  0,
		Sha:       "85f8c029b902ed9400bc600bac301a0aadb144ac",
		Branch:    "master",
		Author:    "drcooper@caltech.edu",
		Gravatar:  "ab23a88a3ed77ecdfeb894c0eaf2817a",
		Timestamp: "Wed Apr 23 03:00:00 2014 -0700",
		Message:   "a commit message",
		Created:   1398065343,
		Updated:   1398065344,
	})
}
