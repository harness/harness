CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE entity_labels (
    entity_label_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_label_label_id INTEGER NOT NULL,
    entity_label_label_value_id INTEGER DEFAULT NULL,
    entity_uuid UUID NOT NULL,
    entity_type TEXT NOT NULL,
    entity_label_created_at BIGINT NOT NULL,
    entity_label_created_by INTEGER NOT NULL,
    entity_label_updated_at BIGINT NOT NULL,
    entity_label_updated_by INTEGER NOT NULL,

    CONSTRAINT fk_entity_labels_label_id FOREIGN KEY (entity_label_label_id)
        REFERENCES labels (label_id) ON DELETE CASCADE,
    CONSTRAINT fk_entity_labels_label_value_id FOREIGN KEY (entity_label_label_value_id)
        REFERENCES label_values (label_value_id) ON DELETE CASCADE,
    
    CONSTRAINT unique_entity_labels UNIQUE (entity_label_label_id, entity_uuid, entity_type)
);


CREATE INDEX idx_entity_labels_uuid_type ON entity_labels (entity_uuid, entity_type);
CREATE INDEX idx_entity_labels_label_value_id ON entity_labels (entity_label_label_value_id);