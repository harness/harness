ALTER TABLE linked_repositories DROP COLUMN linked_repo_provider_repo_id;
ALTER TABLE linked_repositories ADD COLUMN linked_repo_clone_url TEXT NOT NULL DEFAULT '';
