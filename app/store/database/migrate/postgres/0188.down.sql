DROP INDEX pullreqs_root_space_id;
DROP INDEX repositories_root_space_id;
DROP INDEX spaces_root_space_id;

ALTER TABLE pullreqs DROP COLUMN pullreq_root_space_id;
ALTER TABLE repositories DROP COLUMN repo_root_space_id;
ALTER TABLE spaces DROP COLUMN space_root_space_id;
