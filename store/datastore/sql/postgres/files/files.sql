-- name: files-find-build

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

-- name: files-find-proc-name

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

-- name: files-find-proc-name-data

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

-- name: files-delete-build

DELETE FROM files WHERE file_build_id = $1
