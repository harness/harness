CREATE TABLE git_signature_results (
	git_signature_result_repo_id INTEGER NOT NULL,
	git_signature_result_object_sha TEXT NOT NULL,
	git_signature_result_object_time BIGINT NOT NULL,
	git_signature_result_created BIGINT NOT NULL,
	git_signature_result_updated BIGINT NOT NULL,
	git_signature_result_result TEXT NOT NULL,
	git_signature_result_principal_id INTEGER NOT NULL,
	git_signature_result_key_scheme TEXT NOT NULL,
	git_signature_result_key_id TEXT NOT NULL,
	git_signature_result_key_fingerprint TEXT NOT NULL,

	CONSTRAINT pk_git_signature_results PRIMARY KEY (git_signature_result_repo_id, git_signature_result_object_sha),

	CONSTRAINT fk_git_signature_results_repo_id FOREIGN KEY (git_signature_result_repo_id)
		REFERENCES repositories (repo_id)
		ON UPDATE NO ACTION
		ON DELETE CASCADE,

	CONSTRAINT fk_git_signature_result_principal_id FOREIGN KEY (git_signature_result_principal_id)
		REFERENCES principals (principal_id)
		ON UPDATE NO ACTION
		ON DELETE SET NULL
);

CREATE INDEX idx_git_signature_results_principal_id
	ON git_signature_results (git_signature_result_principal_id);
