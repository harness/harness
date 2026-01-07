ALTER TABLE plugins
    ADD COLUMN plugin_type TEXT NOT NULL,
    ADD COLUMN plugin_version TEXT NOT NULL;