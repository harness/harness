package postgres

// Lookup returns the named statement.
func Lookup(name string) string {
	return index[name]
}

var index = map[string]string{
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
}

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
WHERE file_build_id = $1
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
WHERE file_proc_id = $1
  AND file_name    = $2
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
WHERE file_proc_id = $1
  AND file_name    = $2
`

var filesDeleteBuild = `
DELETE FROM files WHERE file_build_id = $1
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
WHERE proc_id = $1
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
WHERE proc_build_id = $1
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
WHERE proc_build_id = $1
  AND proc_pid      = $2
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
WHERE proc_build_id = $1
  AND proc_ppid = $2
  AND proc_name = $3
`

var procsDeleteBuild = `
DELETE FROM procs WHERE proc_build_id = $1
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
WHERE registry_repo_id = $1
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
WHERE registry_repo_id = $1
  AND registry_addr = $2
`

var registryDeleteRepo = `
DELETE FROM registry WHERE registry_repo_id = $1
`

var registryDelete = `
DELETE FROM registry WHERE registry_id = $1
`
