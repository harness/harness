// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mysql

// Lookup returns the named statement.
func Lookup(name string) string {
	return index[name]
}

var index = map[string]string{
	"config-find-id":              configFindId,
	"config-find-repo-hash":       configFindRepoHash,
	"config-find-approved":        configFindApproved,
	"count-users":                 countUsers,
	"count-repos":                 countRepos,
	"count-builds":                countBuilds,
	"feed-latest-build":           feedLatestBuild,
	"feed":                        feed,
	"files-find-build":            filesFindBuild,
	"files-find-proc-name":        filesFindProcName,
	"files-find-proc-name-data":   filesFindProcNameData,
	"files-delete-build":          filesDeleteBuild,
	"logs-find-proc":              logsFindProc,
	"perms-find-user":             permsFindUser,
	"perms-find-user-repo":        permsFindUserRepo,
	"perms-insert-replace":        permsInsertReplace,
	"perms-insert-replace-lookup": permsInsertReplaceLookup,
	"perms-delete-user-repo":      permsDeleteUserRepo,
	"perms-delete-user-date":      permsDeleteUserDate,
	"procs-find-id":               procsFindId,
	"procs-find-build":            procsFindBuild,
	"procs-find-build-pid":        procsFindBuildPid,
	"procs-find-build-ppid":       procsFindBuildPpid,
	"procs-delete-build":          procsDeleteBuild,
	"registry-find-repo":          registryFindRepo,
	"registry-find-repo-addr":     registryFindRepoAddr,
	"registry-delete-repo":        registryDeleteRepo,
	"registry-delete":             registryDelete,
	"repo-update-counter":         repoUpdateCounter,
	"repo-find-user":              repoFindUser,
	"repo-insert-ignore":          repoInsertIgnore,
	"repo-delete":                 repoDelete,
	"secret-find-repo":            secretFindRepo,
	"secret-find-repo-name":       secretFindRepoName,
	"secret-delete":               secretDelete,
	"sender-find-repo":            senderFindRepo,
	"sender-find-repo-login":      senderFindRepoLogin,
	"sender-delete-repo":          senderDeleteRepo,
	"sender-delete":               senderDelete,
	"task-list":                   taskList,
	"task-delete":                 taskDelete,
	"user-find":                   userFind,
	"user-find-login":             userFindLogin,
	"user-update":                 userUpdate,
	"user-delete":                 userDelete,
}

var configFindId = `
SELECT
 config_id
,config_repo_id
,config_hash
,config_data
FROM config
WHERE config_id = ?
`

var configFindRepoHash = `
SELECT
 config_id
,config_repo_id
,config_hash
,config_data
FROM config
WHERE config_repo_id = ?
  AND config_hash    = ?
`

var configFindApproved = `
SELECT build_id FROM builds
WHERE build_repo_id = ?
AND build_config_id = ?
AND build_status NOT IN ('blocked', 'pending')
LIMIT 1
`

var countUsers = `
SELECT count(1)
FROM users
`

var countRepos = `
SELECT count(1)
FROM repos
WHERE repo_active = true
`

var countBuilds = `
SELECT count(1)
FROM builds
`

var feedLatestBuild = `
SELECT
 repo_owner
,repo_name
,repo_full_name
,build_number
,build_event
,build_status
,build_created
,build_started
,build_finished
,build_commit
,build_branch
,build_ref
,build_refspec
,build_remote
,build_title
,build_message
,build_author
,build_email
,build_avatar
FROM repos LEFT OUTER JOIN builds ON build_id = (
	SELECT build_id FROM builds
	WHERE builds.build_repo_id = repos.repo_id
	ORDER BY build_id DESC
	LIMIT 1
)
INNER JOIN perms ON perms.perm_repo_id = repos.repo_id
WHERE perms.perm_user_id = ?
  AND repos.repo_active = true
ORDER BY repo_full_name ASC;
`

var feed = `
SELECT
 repo_owner
,repo_name
,repo_full_name
,build_number
,build_event
,build_status
,build_created
,build_started
,build_finished
,build_commit
,build_branch
,build_ref
,build_refspec
,build_remote
,build_title
,build_message
,build_author
,build_email
,build_avatar
FROM repos
INNER JOIN perms  ON perms.perm_repo_id   = repos.repo_id
INNER JOIN builds ON builds.build_repo_id = repos.repo_id
WHERE perms.perm_user_id = ?
ORDER BY build_id DESC
LIMIT 50
`

var filesFindBuild = `
SELECT
 file_id
,file_build_id
,file_proc_id
,file_pid
,file_name
,file_mime
,file_size
,file_time
,file_meta_passed
,file_meta_failed
,file_meta_skipped
FROM files
WHERE file_build_id = ?
`

var filesFindProcName = `
SELECT
 file_id
,file_build_id
,file_proc_id
,file_pid
,file_name
,file_mime
,file_size
,file_time
,file_meta_passed
,file_meta_failed
,file_meta_skipped
FROM files
WHERE file_proc_id = ?
  AND file_name    = ?
`

var filesFindProcNameData = `
SELECT
 file_id
,file_build_id
,file_proc_id
,file_pid
,file_name
,file_mime
,file_size
,file_time
,file_meta_passed
,file_meta_failed
,file_meta_skipped
,file_data
FROM files
WHERE file_proc_id = ?
  AND file_name    = ?
`

var filesDeleteBuild = `
DELETE FROM files WHERE file_build_id = ?
`

var logsFindProc = `
SELECT
 log_id
,log_job_id
,log_data
FROM logs
WHERE log_job_id = ?
LIMIT 1
`

var permsFindUser = `
SELECT
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_date
FROM perms
WHERE perm_user_id = ?
`

var permsFindUserRepo = `
SELECT
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_synced
FROM perms
WHERE perm_user_id = ?
  AND perm_repo_id = ?
`

var permsInsertReplace = `
REPLACE INTO perms (
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_synced
) VALUES (?,?,?,?,?,?)
`

var permsInsertReplaceLookup = `
REPLACE INTO perms (
 perm_user_id
,perm_repo_id
,perm_pull
,perm_push
,perm_admin
,perm_synced
) VALUES (?,(SELECT repo_id FROM repos WHERE repo_full_name = ?),?,?,?,?)
`

var permsDeleteUserRepo = `
DELETE FROM perms
WHERE perm_user_id = ?
  AND perm_repo_id = ?
`

var permsDeleteUserDate = `
DELETE FROM perms
WHERE perm_user_id = ?
  AND perm_synced < ?
`

var procsFindId = `
SELECT
 proc_id
,proc_build_id
,proc_pid
,proc_ppid
,proc_pgid
,proc_name
,proc_state
,proc_error
,proc_exit_code
,proc_started
,proc_stopped
,proc_machine
,proc_platform
,proc_environ
FROM procs
WHERE proc_id = ?
`

var procsFindBuild = `
SELECT
 proc_id
,proc_build_id
,proc_pid
,proc_ppid
,proc_pgid
,proc_name
,proc_state
,proc_error
,proc_exit_code
,proc_started
,proc_stopped
,proc_machine
,proc_platform
,proc_environ
FROM procs
WHERE proc_build_id = ?
ORDER BY proc_id ASC
`

var procsFindBuildPid = `
SELECT
proc_id
,proc_build_id
,proc_pid
,proc_ppid
,proc_pgid
,proc_name
,proc_state
,proc_error
,proc_exit_code
,proc_started
,proc_stopped
,proc_machine
,proc_platform
,proc_environ
FROM procs
WHERE proc_build_id = ?
  AND proc_pid = ?
`

var procsFindBuildPpid = `
SELECT
proc_id
,proc_build_id
,proc_pid
,proc_ppid
,proc_pgid
,proc_name
,proc_state
,proc_error
,proc_exit_code
,proc_started
,proc_stopped
,proc_machine
,proc_platform
,proc_environ
FROM procs
WHERE proc_build_id = ?
  AND proc_ppid = ?
  AND proc_name = ?
`

var procsDeleteBuild = `
DELETE FROM procs WHERE proc_build_id = ?
`

var registryFindRepo = `
SELECT
 registry_id
,registry_repo_id
,registry_addr
,registry_username
,registry_password
,registry_email
,registry_token
FROM registry
WHERE registry_repo_id = ?
`

var registryFindRepoAddr = `
SELECT
 registry_id
,registry_repo_id
,registry_addr
,registry_username
,registry_password
,registry_email
,registry_token
FROM registry
WHERE registry_repo_id = ?
  AND registry_addr = ?
`

var registryDeleteRepo = `
DELETE FROM registry WHERE registry_repo_id = ?
`

var registryDelete = `
DELETE FROM registry WHERE registry_id = ?
`

var repoUpdateCounter = `
UPDATE repos SET repo_counter = ?
WHERE repo_counter = ?
  AND repo_id = ?
`

var repoFindUser = `
SELECT
 repo_id
,repo_user_id
,repo_owner
,repo_name
,repo_full_name
,repo_avatar
,repo_link
,repo_clone
,repo_branch
,repo_timeout
,repo_private
,repo_trusted
,repo_active
,repo_allow_pr
,repo_allow_push
,repo_allow_deploys
,repo_allow_tags
,repo_hash
,repo_scm
,repo_config_path
,repo_gated
,repo_visibility
,repo_counter
FROM repos
INNER JOIN perms ON perms.perm_repo_id = repos.repo_id
WHERE perms.perm_user_id = ?
ORDER BY repo_full_name ASC
`

var repoInsertIgnore = `
INSERT IGNORE INTO repos (
 repo_user_id
,repo_owner
,repo_name
,repo_full_name
,repo_avatar
,repo_link
,repo_clone
,repo_branch
,repo_timeout
,repo_private
,repo_trusted
,repo_active
,repo_allow_pr
,repo_allow_push
,repo_allow_deploys
,repo_allow_tags
,repo_hash
,repo_scm
,repo_config_path
,repo_gated
,repo_visibility
,repo_counter
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
`

var repoDelete = `
DELETE FROM repos WHERE repo_id = ?
`

var secretFindRepo = `
SELECT
 secret_id
,secret_repo_id
,secret_name
,secret_value
,secret_images
,secret_events
,secret_conceal
,secret_skip_verify
FROM secrets
WHERE secret_repo_id = ?
`

var secretFindRepoName = `
SELECT
secret_id
,secret_repo_id
,secret_name
,secret_value
,secret_images
,secret_events
,secret_conceal
,secret_skip_verify
FROM secrets
WHERE secret_repo_id = ?
  AND secret_name = ?
`

var secretDelete = `
DELETE FROM secrets WHERE secret_id = ?
`

var senderFindRepo = `
SELECT
 sender_id
,sender_repo_id
,sender_login
,sender_allow
,sender_block
FROM senders
WHERE sender_repo_id = ?
`

var senderFindRepoLogin = `
SELECT
 sender_id
,sender_repo_id
,sender_login
,sender_allow
,sender_block
FROM senders
WHERE sender_repo_id = ?
  AND sender_login = ?
`

var senderDeleteRepo = `
DELETE FROM senders WHERE sender_repo_id = ?
`

var senderDelete = `
DELETE FROM senders WHERE sender_id = ?
`

var taskList = `
SELECT
 task_id
,task_data
,task_labels
FROM tasks
`

var taskDelete = `
DELETE FROM tasks WHERE task_id = ?
`

var userFind = `
SELECT
 user_id
,user_login
,user_token
,user_secret
,user_expiry
,user_email
,user_avatar
,user_active
,user_synced
,user_admin
,user_hash
FROM users
ORDER BY user_login ASC
`

var userFindLogin = `
SELECT
 user_id
,user_login
,user_token
,user_secret
,user_expiry
,user_email
,user_avatar
,user_active
,user_synced
,user_admin
,user_hash
FROM users
WHERE user_login = ?
LIMIT 1
`

var userUpdate = `
UPDATE users
SET
,user_token  = ?
,user_secret = ?
,user_expiry = ?
,user_email  = ?
,user_avatar = ?
,user_active = ?
,user_synced = ?
,user_admin  = ?
,user_hash   = ?
WHERE user_id = ?
`

var userDelete = `
DELETE FROM users WHERE user_id = ?
`
