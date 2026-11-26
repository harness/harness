# Testing conventions and coverage program

The coverage target for this repository is 80% (`repo_target` and `patch_target` in the coverage project
settings). This document describes how tests are written here so coverage PRs stay uniform and small.

## PR shape

- Test-only PRs. Do not change production behavior in a coverage PR.
- Around 20 source files per PR. Split large directories, merge tiny ones.
- Aim for at least 80% line coverage on every file a PR adds tests for.
- Run `make test` (or `go test ./path/to/pkg/...`) before pushing.

## Conventions per layer

- Controllers and services: table-driven tests with `testify` (`require`, `mock`). Mock collaborators with the
  hand-written doubles in `mocks/store`, `mocks/git`, `mocks/pullreq` and `mocks/sse`. Add a new double next to the
  existing ones when an interface has none. For tests with more than two or three collaborators, use a fixture
  struct (see `app/api/controller/pullreq/review_submit_test.go`).
- Store and database layer: use the in-process SQLite harness in `app/store/database/setup_test.go`
  (`setupDB`, `setupStores`). Do not mock SQL.
- `git` package: run against real temporary repositories (`t.TempDir()`, `git init`), as `git/parser` and
  `git/hash` do.
- Pure utility and type packages: plain table-driven unit tests.
- Registry: reuse `registry/app/api/controller/mocks`.

## Files excluded from the coverage target

Generated, declarative, test-infrastructure and glue code is excluded from the target instead of being tested.
The patterns below are the `exclude_text` value for the coverage project (glob syntax, one pattern per line,
evaluated by HCLI when a report is uploaded):

```
app/api/openapi/**
registry/app/api/openapi/**
**/wire.go
**/wire_gen.go
mocks/**
registry/app/api/controller/mocks/**
cmd/gitness/**
web/dist.go
registry/app/driver/s3-aws/**
registry/app/driver/gcs/**
registry/app/driver/testsuites/**
registry/docs/**
```

Apply with `PATCH /coverage/api/v1/accounts/<account>/repos/<repoId>/settings` and body
`{"exclude_text": "<patterns joined by newlines>"}`, or through the coverage settings page. This needs a token
with permission to edit coverage settings.

## Measuring progress

The coverage service exposes the per-directory breakdown used to plan this work:

```
BASE=https://harness0.harness.io/coverage/api/v1/accounts/<account>
GET $BASE/repos/<repoId>/commits?orgId=<org>&projectId=<project>
GET $BASE/repos/<repoId>/tree/<dir>/?orgId=<org>&projectId=<project>&commit=<sha>
```

Re-pull after each batch of PRs and compare against the roadmap.
