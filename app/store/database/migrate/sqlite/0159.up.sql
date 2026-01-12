ALTER TABLE repositories ADD column repo_language TEXT DEFAULT '';

CREATE TABLE repo_languages (
  repo_lang_repo_id    INTEGER NOT NULL,
  repo_lang_language  TEXT NOT NULL,
  repo_lang_bytes      INTEGER NOT NULL DEFAULT 0,
  repo_lang_files      INTEGER NOT NULL DEFAULT 0,

  CONSTRAINT repo_languages_pk
    PRIMARY KEY (repo_lang_repo_id, repo_lang_language),

  CONSTRAINT repo_languages_repo_fk FOREIGN KEY (repo_lang_repo_id)
    REFERENCES repositories (repo_id) ON DELETE CASCADE
);
