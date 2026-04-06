-- name: GetOverviewStats :one
SELECT
    (SELECT COUNT(*)::bigint FROM app.projects) AS total_projects,
    (SELECT COUNT(*)::bigint FROM app.nodes) AS total_nodes,
    (SELECT COUNT(*)::bigint FROM app.nodes WHERE current_state = 'online') AS nodes_online,
    (SELECT COUNT(*)::bigint FROM app.nodes WHERE current_state = 'offline') AS nodes_offline,
    (SELECT COUNT(*)::bigint FROM app.services) AS total_services,
    (SELECT COUNT(*)::bigint FROM app.services WHERE current_state = 'healthy') AS services_healthy,
    (SELECT COUNT(*)::bigint FROM app.services WHERE current_state = 'degraded') AS services_degraded,
    (SELECT COUNT(*)::bigint FROM app.services WHERE current_state = 'unhealthy') AS services_unhealthy,
    (SELECT COUNT(*)::bigint FROM app.alert_instances WHERE status = 'active') AS active_alerts,
    (SELECT COUNT(*)::bigint FROM app.alert_instances WHERE status = 'resolved') AS resolved_alerts;

-- name: GetOverviewStatsByProject :one
SELECT
    (SELECT COUNT(*)::bigint FROM app.projects AS p WHERE p.id = $1) AS total_projects,
    (SELECT COUNT(*)::bigint FROM app.nodes AS n WHERE n.project_id = $1) AS total_nodes,
    (SELECT COUNT(*)::bigint FROM app.nodes AS n WHERE n.project_id = $1 AND n.current_state = 'online') AS nodes_online,
    (SELECT COUNT(*)::bigint FROM app.nodes AS n WHERE n.project_id = $1 AND n.current_state = 'offline') AS nodes_offline,
    (SELECT COUNT(*)::bigint FROM app.services AS s WHERE s.project_id = $1) AS total_services,
    (SELECT COUNT(*)::bigint FROM app.services AS s WHERE s.project_id = $1 AND s.current_state = 'healthy') AS services_healthy,
    (SELECT COUNT(*)::bigint FROM app.services AS s WHERE s.project_id = $1 AND s.current_state = 'degraded') AS services_degraded,
    (SELECT COUNT(*)::bigint FROM app.services AS s WHERE s.project_id = $1 AND s.current_state = 'unhealthy') AS services_unhealthy,
    (SELECT COUNT(*)::bigint FROM app.alert_instances AS ai WHERE ai.project_id = $1 AND ai.status = 'active') AS active_alerts,
    (SELECT COUNT(*)::bigint FROM app.alert_instances AS ai WHERE ai.project_id = $1 AND ai.status = 'resolved') AS resolved_alerts;
