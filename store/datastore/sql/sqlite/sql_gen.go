package sqlite

// Lookup returns the named statement.
func Lookup(name string) string {
	return index[name]
}

var index = map[string]string{
	"files-find-build":          filesFindBuild,
	"files-find-proc-name":      filesFindProcName,
	"files-find-proc-name-data": filesFindProcNameData,
	"procs-find-id":             procsFindId,
	"procs-find-build":          procsFindBuild,
	"procs-find-build-pid":      procsFindBuildPid,
	"procs-find-build-ppid":     procsFindBuildPpid,
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
