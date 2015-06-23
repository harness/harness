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
func (db *Agentstore) Agent(build *types.Build) (string, error) {
	agent, err := getAgent(db, rebind(stmtAgentSelectAgentCommit), build.ID)
	if err != nil {
		return "", err
	}
	return agent.Addr, nil
}

// SetAgent updates an agent in the datastore.
func (db *Agentstore) SetAgent(build *types.Build, addr string) error {
	agent := Agent{Addr: addr, BuildID: build.ID}
	return createAgent(db, rebind(stmtAgentInsert), &agent)
}

type Agent struct {
	ID      int64
	Addr    string
	BuildID int64 `sql:"unique:ux_agent_build"`
}
