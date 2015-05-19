package builtin

import (
	"database/sql"

	common "github.com/drone/drone/pkg/types"
	"github.com/russross/meddler"
)

type Agentstore struct {
	*sql.DB
}

func NewAgentstore(db *sql.DB) *Agentstore {
	return &Agentstore{db}
}

// Agent returns an agent by ID.
func (db *Agentstore) Agent(commit *common.Commit) (string, error) {
	var agent = new(agent)
	var err = meddler.QueryRow(db, agent, rebind(agentQuery), commit.ID)
	return agent.Addr, err
}

// SetAgent updates an agent in the datastore.
func (db *Agentstore) SetAgent(commit *common.Commit, addr string) error {
	agent := &agent{}
	agent.Addr = addr
	agent.CommitID = commit.ID
	db.Exec(rebind(deleteAgentQuery), commit.ID)
	return meddler.Insert(db, agentTable, agent)
}

type agent struct {
	ID       int64  `meddler:"agent_id,pk"`
	Addr     string `meddler:"agent_addr"`
	CommitID int64  `meddler:"commit_id"`
}

// Build table name in database.
const agentTable = "agents"

const agentQuery = `
SELECT *
FROM agents
WHERE commit_id = ?
LIMIT 1;
`

const deleteAgentQuery = `
DELETE FROM agents
WHERE commit_id = ?;
`
