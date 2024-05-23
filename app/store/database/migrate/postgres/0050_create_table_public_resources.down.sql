ALTER TABLE repositories ADD COLUMN repo_is_public BOOLEAN NOT NULL DEFAULT FALSE;

-- update repo public access
UPDATE repositories
SET repo_is_public = TRUE
WHERE repo_id IN (
    SELECT public_access_repo_id
    FROM public_access_repo
);

ALTER TABLE spaces ADD COLUMN space_is_public BOOLEAN NOT NULL DEFAULT FALSE;

-- update space public access
UPDATE spaces
SET space_is_public = TRUE
WHERE space_id IN (
    SELECT public_access_space_id
    FROM public_access_space
);

-- clear public access
DROP TABLE public_access_repo;
DROP TABLE public_access_space