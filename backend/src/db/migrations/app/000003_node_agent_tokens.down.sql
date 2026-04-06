DROP INDEX IF EXISTS app.idx_nodes_agent_token_hash;

ALTER TABLE app.nodes
    DROP COLUMN IF EXISTS agent_token_hash;
