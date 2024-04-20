CREATE TABLE public_resources (
 public_resource_id SERIAL PRIMARY KEY
,public_resource_type TEXT NOT NULL
,public_resource_space_id INTEGER
,public_resource_repo_id INTEGER

,CONSTRAINT fk_public_resource_space_id FOREIGN KEY (public_resource_space_id)
    REFERENCES spaces (space_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_public_resource_repo_id FOREIGN KEY (public_resource_repo_id)
    REFERENCES repositories (repo_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);

CREATE UNIQUE INDEX public_resource_space_id_key
	ON public_resources(public_resource_space_id)
	WHERE public_resource_space_id IS NOT NULL;

CREATE UNIQUE INDEX public_resource_repo_id_key
	ON public_resources(public_resource_repo_id)
	WHERE public_resource_repo_id IS NOT NULL;

-- move public repos into public_resource
INSERT INTO public_resources (
     public_resource_type
    ,public_resource_space_id
    ,public_resource_repo_id
)
SELECT
     'repository'
    ,NULL
    ,repo_id
FROM repositories
WHERE repo_is_public = TRUE;

-- alter repo table
ALTER TABLE repositories DROP COLUMN repo_is_public;

-- move public spaces into public_resource
INSERT INTO public_resources (
     public_resource_type
    ,public_resource_space_id
    ,public_resource_repo_id
)
SELECT
     'space'
    ,space_id
    ,NULL
FROM spaces
WHERE space_is_public = TRUE;

-- alter space table
ALTER TABLE spaces DROP COLUMN space_is_public;
