# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.1] - 2019-08-26
### Added

- support for the GitHub deployment status API, by [@bradrydzewski](https://github.com/bradrydzewski).

## [1.3.0] - 2019-08-20
### Added
- support for storing logs in Azure Cloud Storage, by [@Lucretius](https://github.com/Lucretius). [#2788](https://github.com/drone/drone/pull/2788)
- support for windows server 1903, by [@bradrydzewski](https://github.com/bradrydzewski).
- button to view the full log file, by [@dramich](https://github.com/dramich). [drone/drone-ui#287](https://github.com/drone/drone-ui/pull/287).

### Fixed
- read gogs sha from webhook, by [@marcotuna](https://github.com/marcotuna).
- create bind volume on host if not exists, by [@bradrydzewski](https://github.com/bradrydzewski). [#2725](https://github.com/drone/drone/issues/2725).
- preserve whitespace in build logs, by [@geek1011](https://github.com/geek1011). [drone/drone-ui#294](https://github.com/drone/drone-ui/pull/294).
- enable log file download on firefox, by [@bobmanary](https://github.com/bobmanary). [drone/drone-ui#303](https://github.com/drone/drone-ui/pull/303)

### Security
- upgraded to Go 1.12.9 due to CVE-2019-9512 and CVE-2019-9514

## [1.2.3] - 2019-07-30
### Added

- disable github status for cron jobs
- support for action in conditionals, by [@bradrydzewski](https://github.com/bradrydzewski). [#2685](https://github.com/drone/drone/issues/2685).

### Fixed

- improve cancel logic for dangling stages, by [@bradrydzewski](https://github.com/bradrydzewski).
- improve error when kubernetes malforms the port configuration, by [@bradrydzewski](https://github.com/bradrydzewski). [#2742](https://github.com/drone/drone/issues/2742).
- copy parameters from parent build when promoting, by [@bradrydzewski](https://github.com/bradrydzewski). [#2748](https://github.com/drone/drone/issues/2748).

## [1.2.2] - 2019-07-29
### Added

- support for legacy environment variables
- support for legacy workspace based on repository name
- support for github deployment hooks
- provide base sha for github pull requests
- option to filter webhooks by event and type
- upgrade drone-yaml to v1.2.2
- upgrade drone-runtime to v1.0.7

### Fixed

- error when manually creating an empty user, by [@bradrydzewski](https://github.com/bradrydzewski). [#2738](https://github.com/drone/drone/issues/2738).

## [1.2.1] - 2019-06-11
### Added

- support for legacy tokens to ease upgrade path, by [@bradrydzewski](https://github.com/bradrydzewski). [#2713](https://github.com/drone/drone/issues/2713).
- include repository name and id in batch update error message, by [@bradrydzewski](https://github.com/bradrydzewski).

### Fixed

- fix inconsistent base64 encoding and decoding of encrypted secrets, by [@bradrydzewski](https://github.com/bradrydzewski).
- update drone-yaml to version 1.1.2 for improved 0.8 to 1.0 yaml marshal escaping.
- update drone-yaml to version 1.1.3 for improved 0.8 to 1.0 workspace conversion.

## [1.2.0] - 2019-05-30
### Added

- endpoint to trigger new build for default branch, by [@bradrydzewski](https://github.com/bradrydzewski). [#2679](https://github.com/drone/drone/issues/2679).
- endpoint to trigger new build for branch, by [@bradrydzewski](https://github.com/bradrydzewski). [#2679](https://github.com/drone/drone/issues/2679).
- endpoint to trigger new build for branch and sha, by [@bradrydzewski](https://github.com/bradrydzewski). [#2679](https://github.com/drone/drone/issues/2679).
- enable optional prometheus metrics guest access, by [@janberktold](https://github.com/janberktold)
- fallback to database when logs not found in s3, by [@bradrydzewski](https://github.com/bradrydzewski). [#2689](https://github.com/drone/drone/issues/2689).
- support for custom stage definitions and runners, by [@bradrydzewski](https://github.com/bradrydzewski). [#2680](https://github.com/drone/drone/issues/2680).
- update drone-yaml to version 1.1.0

### Fixed

- retrieve latest build by branch, by [@tboerger](https://github.com/tboerger).
- copy the fork value when restarting a build, by [@bradrydzewski](https://github.com/bradrydzewski). [#2708](https://github.com/drone/drone/issues/2708).
- make healthz available without redirect, by [@bradrydzewski](https://github.com/bradrydzewski). [#2706](https://github.com/drone/drone/issues/2706).

## [1.1.0] - 2019-04-23
### Added

- specify a user for the pipeline step, by [@bradrydzewski](https://github.com/bradrydzewski). [#2651](https://github.com/drone/drone/issues/2651).
- support for Gitea oauth2, by [@techknowlogick](https://github.com/techknowlogick). [#2622](https://github.com/drone/drone/pull/2622).
- ping the docker daemon before starting the agent, by [@bradrydzewski](https://github.com/bradrydzewski). [#2495](https://github.com/drone/drone/issues/2495).
- support for Cron job name in Yaml trigger block, by [@bradrydzewski](https://github.com/bradrydzewski). [#2628](https://github.com/drone/drone/issues/2628).
- support for Cron job name in Yaml when block, by [@bradrydzewski](https://github.com/bradrydzewski). [#2628](https://github.com/drone/drone/issues/2628).
- sqlite username column changed to case-insensitive, by [@bradrydzewski](https://github.com/bradrydzewski).
- endpoint to purge repository from database, by [@bradrydzewski](https://github.com/bradrydzewski).
- support for per-organization secrets, by [@bradrydzewski](https://github.com/bradrydzewski).
- include system metadata in global webhooks, by [@bradrydzewski](https://github.com/bradrydzewski).
- ability to customize cookie secure flag, by [@bradrydzewski](https://github.com/bradrydzewski).
- update drone-yaml from version 1.0.6 to 1.0.8.
- update drone-runtime from version 1.0.4 to 1.0.6.
- update go-scm from version 1.0.3 to 1.0.4.

### Fixed

- fixed error in mysql table creation syntax, from [@xuyang2](https://github.com/xuyang2). [#2677](https://github.com/drone/drone/pull/2677).
- fixed stuck builds when upstream dependency is skipped, from [@bradrydzewski](https://github.com/bradrydzewski). [#2634](https://github.com/drone/drone/issues/2634).
- fixed issue running steps with dependencies on failure, from [@bradrydzewski](https://github.com/bradrydzewski). [#2667](https://github.com/drone/drone/issues/2667).

## [1.0.1] - 2019-04-10
### Added

- pass stage environment variables to pipeline steps, by [@bradrydzewski](https://github.com/bradrydzewski).
- update go-scm to version 1.3.0, by [@bradrydzewski](https://github.com/bradrydzewski).
- update drone-runtime to version to 1.0.4, by [@bradrydzewski](https://github.com/bradrydzewski).
- ping docker daemon before agent starts to ensure connectivity, by [@bradrydzewski](https://github.com/bradrydzewski).
