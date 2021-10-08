/*
Revision: 20210507080536
Author:   Andrew Pillar <me@andrewpillar.com>

Create database users
*/

CREATE USER djinn_consumer;
CREATE USER djinn_curator;
CREATE USER djinn_server;
CREATE USER djinn_scheduler;
CREATE USER djinn_worker;

GRANT CONNECT
ON DATABASE djinn
TO djinn_consumer, djinn_curator, djinn_scheduler, djinn_server, djinn_worker;

GRANT SELECT, UPDATE ON build_artifacts TO djinn_curator;
GRANT SELECT ON users TO djinn_curator;
GRANT SELECT, INSERT, UPDATE ON build_artifacts TO djinn_curator;

GRANT SELECT ON images TO djinn_consumer;
GRANT UPDATE ON images, image_downloads TO djinn_consumer;

GRANT SELECT ON builds,
	build_artifacts,
	build_drivers,
	build_jobs,
	build_stages,
	build_tags,
	build_triggers,
	cron,
	cron_builds,
	keys,
	namespaces,
	objects,
	variables,
	users TO djinn_scheduler;

GRANT INSERT ON builds,
	build_artifacts,
	build_drivers,
	build_jobs,
	build_keys,
	build_objects,
	build_stages,
	build_tags,
	build_triggers,
	build_variables,
	cron_builds,
	namespaces
	TO djinn_scheduler;

GRANT UPDATE ON cron TO djinn_scheduler;

GRANT USAGE ON SEQUENCE builds_id_seq,
	build_artifacts_id_seq,
	build_drivers_id_seq,
	build_jobs_id_seq,
	build_objects_id_seq,
	build_stages_id_seq,
	build_tags_id_seq,
	build_triggers_id_seq,
	cron_builds_id_seq,
	namespaces_id_seq
	TO djinn_scheduler;

GRANT SELECT ON builds,
	build_artifacts,
	build_drivers,
	build_jobs,
	build_keys,
	build_objects,
	build_stages,
	build_triggers,
	build_variables,
	images,
	namespace_collaborators,
	objects,
	providers,
	provider_repos,
	users
	TO djinn_worker;

GRANT UPDATE ON builds,
	build_artifacts,
	build_jobs,
	build_objects
	TO djinn_worker;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO djinn_server;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO djinn_server;
