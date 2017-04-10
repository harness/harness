package model

type (
	// ApprovalStore persists approved committers to storage.
	ApprovalStore interface {
		Approve(*Repo) error
	}

	// Approval represents an approved committer.
	Approval struct {
		Username  string `json:"username"   meddler:"approval_username"`
		Approved  bool   `json:"approved"   meddler:"approval_approved"`
		Comments  string `json:"comments"   meddler:"approval_comments"`
		CreatedBy string `json:"created_by" meddler:"approval_created_by"`
		CreatedAt int64  `json:"created_at" meddler:"approval_created_at"`
	}
)
