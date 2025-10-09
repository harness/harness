-- Drop existing tables if they exist
DROP TABLE IF EXISTS quarantined_events;
DROP TABLE IF EXISTS quarantined_paths;

-- Create quarantined_paths table with node_id instead of file_path
CREATE TABLE IF NOT EXISTS quarantined_paths (
    quarantined_path_id TEXT PRIMARY KEY,
    quarantined_path_registry_id INTEGER NOT NULL,
    quarantined_path_image_id INTEGER REFERENCES images (image_id) ON DELETE CASCADE,
    quarantined_path_artifact_id INTEGER REFERENCES artifacts (artifact_id) ON DELETE CASCADE,
    quarantined_path_node_id TEXT REFERENCES nodes (node_id) ON DELETE CASCADE,
    quarantined_path_reason TEXT NOT NULL,
    quarantined_path_created_by INTEGER NOT NULL,
    quarantined_path_created_at INTEGER NOT NULL,
    FOREIGN KEY (quarantined_path_registry_id) REFERENCES registries (registry_id) ON DELETE CASCADE,
    UNIQUE (quarantined_path_registry_id, quarantined_path_artifact_id, quarantined_path_image_id,
            quarantined_path_node_id)
);

-- Create quarantined_events table with node_id instead of file_path
CREATE TABLE IF NOT EXISTS quarantined_events (
    quarantined_path_id TEXT,
    rev_type INTEGER NOT NULL,
    quarantined_event_registry_id INTEGER NOT NULL,
    quarantined_event_image_id INTEGER,
    quarantined_event_artifact_id INTEGER,
    quarantined_event_node_id TEXT,
    quarantined_event_reason TEXT NOT NULL,
    quarantined_event_created_by INTEGER NOT NULL,
    quarantined_event_created_at INTEGER NOT NULL,
    FOREIGN KEY (quarantined_event_registry_id) REFERENCES registries (registry_id) ON DELETE CASCADE
);
