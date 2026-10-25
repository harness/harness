CREATE TABLE shield_scans (
    shield_scan_id             INTEGER PRIMARY KEY AUTOINCREMENT,
    shield_scan_repo_id        INTEGER NOT NULL,
    shield_scan_commit_sha     TEXT NOT NULL DEFAULT '',
    shield_scan_ref            TEXT NOT NULL DEFAULT '',
    shield_scan_trigger        TEXT NOT NULL,
    shield_scan_scanner        TEXT NOT NULL,
    shield_scan_status         TEXT NOT NULL,
    shield_scan_error          TEXT NOT NULL DEFAULT '',
    shield_scan_packages_count INTEGER NOT NULL DEFAULT 0,
    shield_scan_findings_count INTEGER NOT NULL DEFAULT 0,
    shield_scan_started        BIGINT NOT NULL DEFAULT 0,
    shield_scan_finished       BIGINT DEFAULT NULL,
    shield_scan_created        BIGINT NOT NULL,

    CONSTRAINT fk_shield_scans_repo_id FOREIGN KEY (shield_scan_repo_id)
        REFERENCES repositories (repo_id) ON DELETE CASCADE
);

CREATE INDEX shield_scans_repo_id_created ON shield_scans (shield_scan_repo_id, shield_scan_created);

CREATE TABLE shield_findings (
    shield_finding_id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    shield_finding_repo_id            INTEGER NOT NULL,
    shield_finding_fingerprint        TEXT NOT NULL,
    shield_finding_scanner            TEXT NOT NULL,
    shield_finding_kind               TEXT NOT NULL,
    shield_finding_ecosystem          TEXT NOT NULL DEFAULT '',
    shield_finding_package            TEXT NOT NULL DEFAULT '',
    shield_finding_installed_version  TEXT NOT NULL DEFAULT '',
    shield_finding_manifest_path      TEXT NOT NULL DEFAULT '',
    shield_finding_vuln_id            TEXT NOT NULL DEFAULT '',
    shield_finding_aliases            TEXT NOT NULL DEFAULT '[]',
    shield_finding_summary            TEXT NOT NULL DEFAULT '',
    shield_finding_cvss_vector        TEXT NOT NULL DEFAULT '',
    shield_finding_cvss_score         DOUBLE PRECISION NOT NULL DEFAULT 0,
    shield_finding_epss               DOUBLE PRECISION NOT NULL DEFAULT 0,
    shield_finding_in_kev             BOOLEAN NOT NULL DEFAULT FALSE,
    shield_finding_reachable          TEXT NOT NULL DEFAULT 'unknown',
    shield_finding_risk_score         INTEGER NOT NULL DEFAULT 0,
    shield_finding_fix_strategy       TEXT DEFAULT NULL,
    shield_finding_min_fixed_version  TEXT NOT NULL DEFAULT '',
    shield_finding_fixed_versions     TEXT NOT NULL DEFAULT '[]',
    shield_finding_state              TEXT NOT NULL,
    shield_finding_state_reason       TEXT NOT NULL DEFAULT '',
    shield_finding_first_seen_scan_id INTEGER NOT NULL DEFAULT 0,
    shield_finding_last_seen_scan_id  INTEGER NOT NULL DEFAULT 0,
    shield_finding_first_seen         BIGINT NOT NULL,
    shield_finding_last_seen          BIGINT NOT NULL,
    shield_finding_fixed_at           BIGINT DEFAULT NULL,
    shield_finding_task_id            INTEGER DEFAULT NULL,
    shield_finding_ai_decision_ids    TEXT NOT NULL DEFAULT '[]',
    shield_finding_created            BIGINT NOT NULL,
    shield_finding_updated            BIGINT NOT NULL,
    shield_finding_version            BIGINT NOT NULL DEFAULT 0,

    CONSTRAINT fk_shield_findings_repo_id FOREIGN KEY (shield_finding_repo_id)
        REFERENCES repositories (repo_id) ON DELETE CASCADE,
    CONSTRAINT fk_shield_findings_task_id FOREIGN KEY (shield_finding_task_id)
        REFERENCES tasks (task_id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX shield_findings_repo_id_fingerprint ON shield_findings (shield_finding_repo_id, shield_finding_fingerprint);
CREATE INDEX shield_findings_repo_id_state ON shield_findings (shield_finding_repo_id, shield_finding_state);
CREATE INDEX shield_findings_repo_id_risk_score ON shield_findings (shield_finding_repo_id, shield_finding_risk_score);
CREATE INDEX shield_findings_task_id ON shield_findings (shield_finding_task_id);
