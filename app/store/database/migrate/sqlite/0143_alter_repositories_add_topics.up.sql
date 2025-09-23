ALTER TABLE repositories ADD COLUMN repo_topics JSONB NOT NULL DEFAULT '[]';
