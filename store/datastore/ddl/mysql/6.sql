-- +migrate Up

CREATE TABLE agents (
 agent_id       INTEGER PRIMARY KEY AUTO_INCREMENT
,agent_addr     VARCHAR(255)
,agent_platform VARCHAR(500)
,agent_capacity INTEGER
,agent_created  INTEGER
,agent_updated  INTEGER

,UNIQUE(agent_addr)
);


-- +migrate Down

DROP TABLE agents;
