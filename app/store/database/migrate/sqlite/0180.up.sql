ALTER TABLE linked_repositories DROP COLUMN linked_repo_clone_url;
ALTER TABLE linked_repositories ADD COLUMN linked_repo_provider_repo_id TEXT NOT NULL DEFAULT '';
