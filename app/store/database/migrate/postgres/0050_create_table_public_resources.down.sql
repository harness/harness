-- copy public repositories
ALTER TABLE repositories ADD COLUMN repo_is_public;

UPDATE repositories
WHERE repo_id IN (
    SELECT public_resource_repo_id
    FROM public_resources 
    WHERE public_resource_repo_id IS NOT NULL;
) SET 
repo_is_public = TRUE;


-- copy public spaces
ALTER TABLE spaces ADD COLUMN space_is_public;

-- update public resources
UPDATE spaces
WHERE space_id IN (
    SELECT public_resource_space_id
    FROM public_resources 
    WHERE public_resource_space_id IS NOT NULL;
) SET 
sapce_is_public = TRUE;

-- clear public_resoureces
DROP INDEX public_resource_space_id_key;
DROP INDEX public_resource_repo_id_key;
DROP TABLE public_resources;