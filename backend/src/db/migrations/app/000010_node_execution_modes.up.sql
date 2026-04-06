ALTER TABLE app.nodes
ADD COLUMN node_type TEXT,
ADD COLUMN is_visible BOOLEAN;

UPDATE app.nodes
SET node_type = 'agent',
    is_visible = TRUE
WHERE node_type IS NULL
   OR is_visible IS NULL;

ALTER TABLE app.nodes
ALTER COLUMN node_type SET NOT NULL,
ALTER COLUMN node_type SET DEFAULT 'agent',
ALTER COLUMN is_visible SET NOT NULL,
ALTER COLUMN is_visible SET DEFAULT TRUE;

ALTER TABLE app.nodes
ADD CONSTRAINT nodes_node_type_check
CHECK (node_type IN ('managed', 'agent'));

CREATE UNIQUE INDEX idx_nodes_one_managed_per_project
ON app.nodes(project_id)
WHERE node_type = 'managed';
