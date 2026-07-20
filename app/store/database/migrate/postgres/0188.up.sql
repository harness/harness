ALTER TABLE spaces ADD COLUMN space_root_space_id INTEGER;
ALTER TABLE repositories ADD COLUMN repo_root_space_id INTEGER;
ALTER TABLE pullreqs ADD COLUMN pullreq_root_space_id INTEGER;

-- backfill spaces: root spaces point to themselves, children walk the hierarchy
WITH RECURSIVE ancestors AS (
    SELECT space_id AS root_id, space_id, space_parent_id
    FROM spaces
    WHERE space_parent_id IS NULL
    UNION ALL
    SELECT a.root_id, s.space_id, s.space_parent_id
    FROM spaces s
    JOIN ancestors a ON s.space_parent_id = a.space_id
)
UPDATE spaces
SET space_root_space_id = ancestors.root_id
FROM ancestors
WHERE spaces.space_id = ancestors.space_id;

-- backfill repositories via their parent space
UPDATE repositories
SET repo_root_space_id = spaces.space_root_space_id
FROM spaces
WHERE spaces.space_id = repositories.repo_parent_id;

-- backfill pullreqs via their target repository
UPDATE pullreqs
SET pullreq_root_space_id = repositories.repo_root_space_id
FROM repositories
WHERE repositories.repo_id = pullreqs.pullreq_target_repo_id;

ALTER TABLE spaces ALTER COLUMN space_root_space_id SET NOT NULL;
ALTER TABLE repositories ALTER COLUMN repo_root_space_id SET NOT NULL;
ALTER TABLE pullreqs ALTER COLUMN pullreq_root_space_id SET NOT NULL;

CREATE INDEX spaces_root_space_id ON spaces(space_root_space_id);
CREATE INDEX repositories_root_space_id ON repositories(repo_root_space_id);
CREATE INDEX pullreqs_root_space_id ON pullreqs(pullreq_root_space_id);
