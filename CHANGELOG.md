# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased
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
- update drone-yaml from version 1.0.6 to 1.0.8.
- update drone-runtime from version 1.0.4 to 1.0.6.
- update go-scm from version 1.0.3 to 1.0.4.

## [1.0.1] - 2019-04-10
### Added

- pass stage environment variables to pipeline steps, by [@bradrydzewski](https://github.com/bradrydzewski).
- update go-scm to version 1.3.0, by [@bradrydzewski](https://github.com/bradrydzewski).
- update drone-runtime to version to 1.0.4, by [@bradrydzewski](https://github.com/bradrydzewski).
- ping docker daemon before agent starts to ensure connectivity, by [@bradrydzewski](https://github.com/bradrydzewski).
