CREATE TABLE ai_tasks (
                         aitask_id INTEGER PRIMARY KEY AUTOINCREMENT,
                         aitask_uid TEXT NOT NULL,
                         aitask_gitspace_config_id INTEGER NOT NULL,
                         aitask_gitspace_instance_id INTEGER,
                         aitask_initial_prompt TEXT NOT NULL,
                         aitask_display_name TEXT NOT NULL,
                         aitask_user_uid TEXT NOT NULL,
                         aitask_space_id INTEGER NOT NULL,
                         aitask_created BIGINT NOT NULL,
                         aitask_updated BIGINT NOT NULL,
                         aitask_api_url TEXT,
                         aitask_ai_agent TEXT NOT NULL,
                         aitask_state TEXT NOT NULL,
                          UNIQUE (aitask_uid, aitask_space_id),
                          CONSTRAINT fk_ai_tasks_gitspace_config FOREIGN KEY (aitask_gitspace_config_id)
                              REFERENCES gitspace_configs (gconf_id) MATCH SIMPLE
                              ON UPDATE NO ACTION
                              ON DELETE CASCADE,
                          CONSTRAINT fk_ai_tasks_gitspace_instance FOREIGN KEY (aitask_gitspace_instance_id)
                              REFERENCES gitspaces (gits_id) MATCH SIMPLE
                              ON UPDATE NO ACTION
                              ON DELETE CASCADE,
                          CONSTRAINT fk_ai_tasks_space FOREIGN KEY (aitask_space_id)
                              REFERENCES spaces (space_id) MATCH SIMPLE
                              ON UPDATE NO ACTION
                              ON DELETE CASCADE
);
CREATE UNIQUE INDEX ai_tasks_uid_space_id ON ai_tasks (aitask_uid, aitask_space_id);