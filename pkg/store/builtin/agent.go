package builtin

import (
	"database/sql"

	"github.com/drone/drone/pkg/types"
)

type Agentstore struct {
	*sql.DB
}

func NewAgentstore(db *sql.DB) *Agentstore {
	return &Agentstore{db}
}

// Agent returns an agent by ID.
func (db *Agentstore) Agent(commit *types.Commit) (string, error) {
	agent, err := getAgent(db, rebind(stmtAgentSelectAgentCommit), commit.ID)
	if err != nil {
		return "", err
	}
	return agent.Addr, nil
}

// SetAgent updates an agent in the datastore.
func (db *Agentstore) SetAgent(commit *types.Commit, addr string) error {
	agent := Agent{Addr: addr, CommitID: commit.ID}
	return createAgent(db, rebind(stmtAgentInsert), &agent)
}

type Agent struct {
	ID       int64
	Addr     string
	CommitID int64 `sql:"unique:ux_agent_commit"`
}
