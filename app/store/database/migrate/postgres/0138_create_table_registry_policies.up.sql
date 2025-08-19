CREATE TABLE IF NOT EXISTS registry_policies (
  registry_policy_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  registry_policy_registry_id INTEGER NOT NULL,
  registry_policy_policy_ref TEXT NOT NULL,
  registry_policy_created_at BIGINT NOT NULL,
  registry_policy_created_by BIGINT NOT NULL,
  CONSTRAINT fk_registry_policies_registry_id FOREIGN KEY (registry_policy_registry_id)
    REFERENCES registries (registry_id) ON DELETE CASCADE,
  CONSTRAINT uq_registry_policies_registry_policy UNIQUE (registry_policy_registry_id, registry_policy_policy_ref)
);
