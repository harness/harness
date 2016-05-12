package datastore

import (
	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

func (db *datastore) GetAgent(id int64) (*model.Agent, error) {
	var agent = new(model.Agent)
	var err = meddler.Load(db, agentTable, agent, id)
	return agent, err
}

func (db *datastore) GetAgentAddr(addr string) (*model.Agent, error) {
	var agent = new(model.Agent)
	var err = meddler.QueryRow(db, agent, rebind(agentAddrQuery), addr)
	return agent, err
}

func (db *datastore) GetAgentList() ([]*model.Agent, error) {
	var agents = []*model.Agent{}
	var err = meddler.QueryAll(db, &agents, rebind(agentListQuery))
	return agents, err
}

func (db *datastore) CreateAgent(agent *model.Agent) error {
	return meddler.Insert(db, agentTable, agent)
}

func (db *datastore) UpdateAgent(agent *model.Agent) error {
	return meddler.Update(db, agentTable, agent)
}

func (db *datastore) DeleteAgent(agent *model.Agent) error {
	var _, err = db.Exec(rebind(agentDeleteStmt), agent.ID)
	return err
}

const agentTable = "agents"

const agentAddrQuery = `
SELECT *
FROM agents
WHERE agent_addr=?
LIMIT 1
`

const agentListQuery = `
SELECT *
FROM agents
ORDER BY agent_addr ASC
`

const agentDeleteStmt = `
DELETE FROM agents WHERE agent_id = ?
`
