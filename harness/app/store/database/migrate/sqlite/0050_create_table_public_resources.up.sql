CREATE TABLE public_access_repo (
    public_access_repo_id INTEGER PRIMARY KEY,
    CONSTRAINT fk_public_access_repo_id FOREIGN KEY (public_access_repo_id)
        REFERENCES repositories (repo_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE public_access_space (
    public_access_space_id INTEGER PRIMARY KEY,
    CONSTRAINT fk_public_access_space_id FOREIGN KEY (public_access_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

INSERT INTO public_access_repo (
     public_access_repo_id
)
SELECT
     repo_id
FROM repositories
WHERE repo_is_public = TRUE;

ALTER TABLE repositories DROP COLUMN repo_is_public;

INSERT INTO public_access_space (
     public_access_space_id
)
SELECT
     space_id
FROM spaces
WHERE space_is_public = TRUE;

ALTER TABLE spaces DROP COLUMN space_is_public;
