-- name: create-table-agents

CREATE TABLE IF NOT EXISTS agents (
 agent_id       INTEGER PRIMARY KEY AUTOINCREMENT
,agent_addr     TEXT
,agent_platform TEXT
,agent_capacity INTEGER
,agent_created  INTEGER
,agent_updated  INTEGER

,UNIQUE(agent_addr)
);
