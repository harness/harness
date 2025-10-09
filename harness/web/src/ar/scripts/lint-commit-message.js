/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const COMMIT_MSG = process.argv[2]
const JIRA_TKT_CHECK = process.env.JIRA_TKT_CHECK === 'true'
const COMMIT_REGEX = JIRA_TKT_CHECK
  ? /^(revert: )?(feat|fix|docs|style|refactor|perf|test|chore)(\(.+\))?:\s\[.*-\d*\]: .{1,50}/
  : /^(revert: )?(feat|fix|docs|style|refactor|perf|test|chore)(\(.+\))?: .{1,50}/
const ERROR_MSG = JIRA_TKT_CHECK
  ? 'Commit messages must be "fix/feat/docs/style/refactor/perf/test/chore: [AH-<ticket number>]: <changes>"'
  : 'Commit messages must be "fix/feat/docs/style/refactor/perf/test/chore: <changes>"'
if (!COMMIT_REGEX.test(COMMIT_MSG)) {
  console.log(ERROR_MSG)
  console.log(`But got: "${COMMIT_MSG}"`)
  process.exit(1)
}
