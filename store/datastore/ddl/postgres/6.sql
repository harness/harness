-- +migrate Up

CREATE TABLE agents (
 agent_id       SERIAL PRIMARY KEY
,agent_addr     VARCHAR(500)
,agent_platform VARCHAR(500)
,agent_capacity INTEGER
,agent_created  INTEGER
,agent_updated  INTEGER

,UNIQUE(agent_addr)
);


-- +migrate Down

DROP TABLE agents;
