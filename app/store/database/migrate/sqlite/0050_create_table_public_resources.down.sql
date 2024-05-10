-- copy public repositories
ALTER TABLE repositories ADD COLUMN repo_is_public BOOLEAN;

-- update repo public access
UPDATE repositories
SET repo_is_public = TRUE
WHERE repo_id IN (
    SELECT public_access_repo_id
    FROM public_access
    WHERE public_access_repo_id IS NOT NULL
); 

-- copy public spaces
ALTER TABLE spaces ADD COLUMN space_is_public BOOLEAN;

-- update space public access
UPDATE spaces
SET space_is_public = TRUE
WHERE space_id IN (
    SELECT public_access_space_id
    FROM public_access
    WHERE public_access_space_id IS NOT NULL
); 

-- clear public_access
DROP INDEX public_access_space_id_key;
DROP INDEX public_access_repo_id_key;
DROP TABLE public_access;