package sqlite

// Lookup returns the named statement.
func Lookup(name string) string {
	return index[name]
}

var index = map[string]string{
	"config-find-id":            configFindId,
	"config-find-repo-hash":     configFindRepoHash,
	"config-find-approved":      configFindApproved,
	"count-users":               countUsers,
	"count-repos":               countRepos,
	"count-builds":              countBuilds,
	"files-find-build":          filesFindBuild,
	"files-find-proc-name":      filesFindProcName,
	"files-find-proc-name-data": filesFindProcNameData,
	"files-delete-build":        filesDeleteBuild,
	"procs-find-id":             procsFindId,
	"procs-find-build":          procsFindBuild,
	"procs-find-build-pid":      procsFindBuildPid,
	"procs-find-build-ppid":     procsFindBuildPpid,
	"procs-delete-build":        procsDeleteBuild,
	"registry-find-repo":        registryFindRepo,
	"registry-find-repo-addr":   registryFindRepoAddr,
	"registry-delete-repo":      registryDeleteRepo,
	"registry-delete":           registryDelete,
	"repo-update-counter":       repoUpdateCounter,
	"secret-find-repo":          secretFindRepo,
	"secret-find-repo-name":     secretFindRepoName,
	"secret-delete":             secretDelete,
	"sender-find-repo":          senderFindRepo,
	"sender-find-repo-login":    senderFindRepoLogin,
	"sender-delete-repo":        senderDeleteRepo,
	"sender-delete":             senderDelete,
	"task-list":                 taskList,
	"task-delete":               taskDelete,
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
`

var countBuilds = `
SELECT count(1)
FROM builds
`

var filesFindBuild = `
SELECT
 file_id
,file_build_id
,file_proc_id
,file_name
,file_mime
,file_size
,file_time
FROM files
WHERE file_build_id = ?
`

var filesFindProcName = `
SELECT
 file_id
,file_build_id
,file_proc_id
,file_name
,file_mime
,file_size
,file_time
FROM files
WHERE file_proc_id = ?
  AND file_name    = ?
`

var filesFindProcNameData = `
SELECT
 file_id
,file_build_id
,file_proc_id
,file_name
,file_mime
,file_size
,file_time
,file_data
FROM files
WHERE file_proc_id = ?
  AND file_name    = ?
`

var filesDeleteBuild = `
DELETE FROM files WHERE file_build_id = ?
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
