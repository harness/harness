CREATE UNIQUE INDEX settings_sys_key
	ON settings(LOWER(setting_key))
	WHERE setting_repo_id IS NULL AND setting_space_id IS NULL;