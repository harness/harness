UPDATE infra_provisioned
SET iprov_infra_status = 'stopped'
WHERE iprov_gitspace_id IN (SELECT gitspaces.gits_id from gitspaces where gits_state = 'cleaning');

UPDATE gitspaces
SET gits_state = 'cleaned'
WHERE gits_state = 'cleaning';