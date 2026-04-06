DROP INDEX IF EXISTS idx_nodes_one_managed_per_project;

ALTER TABLE app.nodes
DROP CONSTRAINT IF EXISTS nodes_node_type_check;

ALTER TABLE app.nodes
DROP COLUMN IF EXISTS is_visible,
DROP COLUMN IF EXISTS node_type;
