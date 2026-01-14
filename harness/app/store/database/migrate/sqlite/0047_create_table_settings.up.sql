CREATE TABLE settings (
 setting_id INTEGER PRIMARY KEY AUTOINCREMENT
,setting_space_id INTEGER
,setting_repo_id INTEGER
,setting_key TEXT NOT NULL
,setting_value TEXT

,CONSTRAINT fk_settings_space_id FOREIGN KEY (setting_space_id)
    REFERENCES spaces (space_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_settings_repo_id FOREIGN KEY (setting_repo_id)
    REFERENCES repositories (repo_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);

CREATE UNIQUE INDEX settings_space_id_key
	ON settings(setting_space_id, LOWER(setting_key))
	WHERE setting_space_id IS NOT NULL;

CREATE UNIQUE INDEX settings_repo_id_key
	ON settings(setting_repo_id, LOWER(setting_key))
	WHERE setting_repo_id IS NOT NULL;