ALTER TABLE nodes DROP CONSTRAINT IF EXISTS nodes_node_parent_id_fkey;

ALTER TABLE nodes
ADD CONSTRAINT nodes_node_parent_id_fkey
FOREIGN KEY (node_parent_id) 
REFERENCES nodes(node_id)
ON DELETE CASCADE;