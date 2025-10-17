CREATE TABLE IF NOT EXISTS gitspace_settings (
                                                 gsett_id SERIAL PRIMARY KEY,
                                                 gsett_space_id BIGINT NOT NULL,
                                                 gsett_settings_data JSONB NOT NULL,
                                                 gsett_settings_type TEXT NOT NULL,
                                                 gsett_criteria_key TEXT NOT NULL,
                                                 gsett_created BIGINT NOT NULL,
                                                 gsett_updated BIGINT NOT NULL,
                                                 UNIQUE (gsett_space_id, gsett_settings_type,gsett_criteria_key)
);
