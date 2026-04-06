ALTER TABLE app.nodes
    ADD COLUMN agent_token_hash TEXT;

CREATE UNIQUE INDEX idx_nodes_agent_token_hash
    ON app.nodes(agent_token_hash)
    WHERE agent_token_hash IS NOT NULL;
