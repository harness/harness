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
func (db *Agentstore) Agent(id int64) (*common.Agent, error) {
	var agent = new(common.Agent)
	var err = meddler.Load(db, agentTable, agent, id)
	return agent, err
}

// AgentAddr returns an agent by address.
func (db *Agentstore) AgentAddr(addr string) (*common.Agent, error) {
	var agent = new(common.Agent)
	var err = meddler.QueryRow(db, agent, rebind(agentAddrQuery), addr)
	return agent, err
}

// AgentToken returns an agent by token.
func (db *Agentstore) AgentToken(token string) (*common.Agent, error) {
	var agent = new(common.Agent)
	var err = meddler.QueryRow(db, agent, rebind(agentTokenQuery), token)
	return agent, err
}

// AgentList returns a list of all build agents.
func (db *Agentstore) AgentList() ([]*common.Agent, error) {
	var agents []*common.Agent
	var err = meddler.QueryAll(db, &agents, rebind(agentListQuery), true)
	return agents, err
}

// AddAgent inserts an agent in the datastore.
func (db *Agentstore) AddAgent(agent *common.Agent) error {
	return meddler.Insert(db, agentTable, agent)
}

// SetAgent updates an agent in the datastore.
func (db *Agentstore) SetAgent(agent *common.Agent) error {
	return meddler.Update(db, agentTable, agent)
}

// Agent table name in database.
const agentTable = "agents"

const agentTokenQuery = `
SELECT *
FROM agents
WHERE agent_token = ?
LIMIT 1;
`

const agentAddrQuery = `
SELECT *
FROM agents
WHERE agent_addr = ?
LIMIT 1;
`

const agentListQuery = `
SELECT *
FROM agents
WHERE agent_active = ?;
`
