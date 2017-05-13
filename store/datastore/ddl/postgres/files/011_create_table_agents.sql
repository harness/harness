-- name: create-table-agents

CREATE TABLE IF NOT EXISTS agents (
 agent_id       SERIAL PRIMARY KEY
,agent_addr     VARCHAR(250)
,agent_platform VARCHAR(500)
,agent_capacity INTEGER
,agent_created  INTEGER
,agent_updated  INTEGER

,UNIQUE(agent_addr)
);
