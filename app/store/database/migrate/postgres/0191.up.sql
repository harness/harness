-- Secondary index for inbound-webhook routing (mirrors idx_linked_pullreqs_provider_id
-- added in 0183). ListByProviderID filters on (provider_type, provider_repo_id) for
-- every PR/Check/Push webhook event, across all linked-repo accounts.
CREATE INDEX idx_linked_repositories_provider_id
    ON linked_repositories (linked_repo_provider_type, linked_repo_provider_repo_id);

-- Serves the inbound-webhook presence check (ExistsByRootSpace): "does this root
-- space own any live linked repo?" without scanning repositories. Partial, so it
-- only holds linked repos.
CREATE INDEX idx_repositories_linked_root_space
    ON repositories (repo_root_space_identifier)
    WHERE repo_type = 'linked' AND repo_deleted IS NULL;
