ALTER TABLE checks
    ADD COLUMN check_bypassed_by INTEGER,
    ADD CONSTRAINT fk_check_bypassed_by FOREIGN KEY (check_bypassed_by)
        REFERENCES principals (principal_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE SET NULL;
