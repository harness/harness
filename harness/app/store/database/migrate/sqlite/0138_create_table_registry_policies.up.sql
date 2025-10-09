CREATE TABLE IF NOT EXISTS registry_policies (
  registry_policy_id TEXT PRIMARY KEY,
  registry_policy_registry_id INTEGER NOT NULL,
  registry_policy_policy_ref TEXT NOT NULL,
  registry_policy_created_at INTEGER NOT NULL,
  registry_policy_created_by INTEGER NOT NULL,
  FOREIGN KEY (registry_policy_registry_id) REFERENCES registries (registry_id) ON DELETE CASCADE,
  UNIQUE (registry_policy_registry_id, registry_policy_policy_ref)
);
