ALTER TABLE gitspaces ADD COLUMN gits_active_time_started BIGINT;
ALTER TABLE gitspaces ADD COLUMN gits_active_time_ended BIGINT;

UPDATE gitspaces
SET gits_active_time_started =
        CASE
            WHEN gits_state = 'uninitialized' THEN NULL
            ELSE gits_created
            END;

UPDATE gitspaces
SET gits_active_time_ended =
        CASE
            WHEN gits_state IN ('running', 'starting', 'uninitialized') THEN NULL
            ELSE gits_updated
            END;

UPDATE gitspaces
SET gits_total_time_used = gits_active_time_ended - gits_active_time_started
WHERE gits_active_time_ended IS NOT NULL
  AND gits_active_time_started IS NOT NULL;
