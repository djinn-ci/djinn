

DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS oauth_codes;
DROP TABLE IF EXISTS oauth_apps;
DROP TABLE IF EXISTS provider_repos;
DROP TABLE IF EXISTS providers;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS build_keys;
DROP TABLE IF EXISTS namespace_collaborators;
DROP TABLE IF EXISTS namespace_invites;
DROP TABLE IF EXISTS keys;
DROP TABLE IF EXISTS build_triggers;
DROP TYPE IF EXISTS trigger_type;
DROP TABLE IF EXISTS build_drivers;
DROP TYPE IF EXISTS driver_type;
DROP TABLE IF EXISTS build_variables;
DROP TABLE IF EXISTS variables;
DROP TABLE IF EXISTS build_artifacts;
DROP TABLE IF EXISTS build_objects;
DROP TABLE IF EXISTS objects;
DROP TABLE IF EXISTS build_jobs;
DROP TABLE IF EXISTS build_stages;
DROP TABLE IF EXISTS build_tags;
DROP TABLE IF EXISTS builds;
DROP TYPE IF EXISTS status; 
DROP TABLE IF EXISTS namespaces;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS visibility;
DROP TYPE IF EXISTS status;
DROP TYPE IF EXISTS driver_type;
DROP TYPE IF EXISTS trigger_type;

CREATE TYPE visibility AS ENUM ('private', 'internal', 'public');
CREATE TYPE status AS ENUM ('queued', 'running', 'passed', 'failed', 'passed_with_failures', 'killed', 'timed_out');
CREATE TYPE driver_type AS ENUM ('ssh', 'qemu', 'docker');
CREATE TYPE trigger_type AS ENUM ('manual', 'push', 'pull');

CREATE TABLE users (
	id         SERIAL PRIMARY KEY,
	email      VARCHAR NOT NULL UNIQUE,
	username   VARCHAR NOT NULL UNIQUE,
	password   BYTEA NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	deleted_at TIMESTAMP NULL
);

CREATE TABLE namespaces (
	id          SERIAL PRIMARY KEY,
	user_id     INT NOT NULL REFERENCES users(id),
	root_id     INT NULL,
	parent_id   INT NULL,
	name        VARCHAR NOT NULL,
	path        VARCHAR NOT NULL,
	description VARCHAR NULL,
	level       INT NOT NULL,
	visibility  visibility DEFAULT 'private',
	created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE namespace_invites (
	id           SERIAL PRIMARY KEY,
	namespace_id INT NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
	invitee_id   INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	inviter_id   INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE namespace_collaborators (
	id           SERIAL PRIMARY KEY,
	namespace_id INT NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
	user_id      INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE provider_repos (
	id          SERIAL PRIMARY KEY,
	user_id     INT NOT NULL REFERENCES users(id),
	provider_id INT NOT NULL REFERENCES providers(id),
	hook_id     INT NOT NULL,
	repo_id     INT NOT NULL,
	enabled     BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE images (
	id           SERIAL PRIMARY KEY,
	user_id      INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	namespace_id INT NULL REFERENCES namespaces(id) ON DELETE SET NULL,
	driver       driver_type NOT NULL,
	hash         VARCHAR NOT NULL,
	name         VARCHAR NOT NULL
);

CREATE TABLE objects (
	id           SERIAL PRIMARY KEY,
	user_id      INT NOT NULL REFERENCES users(id),
	namespace_id INT NULL REFERENCES namespaces(id) ON DELETE SET NULL,
	hash         VARCHAR NOT NULL UNIQUE,
	name         VARCHAR NOT NULL,
	type         VARCHAR NOT NULL,
	size         INT NOT NULL,
	md5          BYTEA NOT NULL,
	sha256       BYTEA NOT NULL,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
	deleted_at   TIMESTAMP NULL
);

CREATE TABLE variables (
	id            SERIAL PRIMARY KEY,
	user_id       INT NOT NULL REFERENCES users(id),
	namespace_id  INT NULL REFERENCES namespaces(id) ON DELETE SET NULL,
	key           VARCHAR NOT NULL,
	value         VARCHAR NOT NULL,
	created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE keys (
	id           SERIAL PRIMARY KEY,
	user_id      INT NOT NULL REFERENCES users(id),
	namespace_id INT NULL REFERENCES namespaces(id) ON DELETE SET NULL,
	name         VARCHAR NOT NULL,
	key          BYTEA NOT NULL,
	config       TEXT NOT NULL,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE builds (
	id           SERIAL PRIMARY KEY,
	user_id      INT NOT NULL REFERENCES users(id),
	namespace_id INT NULL REFERENCES namespaces(id) ON DELETE SET NULL,
	manifest     TEXT NOT NULL,
	status       status DEFAULT 'queued',
	output       TEXT NULL,
	secret       TEXT NULL,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
	started_at   TIMESTAMP NULL,
	finished_at  TIMESTAMP NULL
);

CREATE TABLE build_tags (
	id         SERIAL PRIMARY KEY,
	user_id    INT NOT NULL REFERENCES users(id),
	build_id   INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	name       VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE build_stages (
	id          SERIAL PRIMARY KEY,
	build_id    INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	name        VARCHAR NOT NULL,
	can_fail    BOOLEAN NOT NULL,
	status      status DEFAULT 'queued',
	started_at  TIMESTAMP NULL,
	finished_at TIMESTAMP NULL
);

CREATE TABLE build_jobs (
	id          SERIAL PRIMARY KEY,
	build_id    INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	stage_id    INT NOT NULL REFERENCES build_stages(id) ON DELETE CASCADE,
	name        VARCHAR NOT NULL,
	commands    VARCHAR NOT NULL,
	output      TEXT NULL,
	status      status DEFAULT 'queued',
	started_at  TIMESTAMP NULL,
	finished_at TIMESTAMP NULL
);

CREATE TABLE build_objects (
	id          SERIAL PRIMARY KEY,
	build_id    INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	object_id   INT NULL REFERENCES objects(id) ON DELETE SET NULL,
	source      VARCHAR NOT NULL,
	name        VARCHAR NOT NULL,
	placed      BOOLEAN NOT NULL DEFAULT FALSE,
	created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE build_artifacts (
	id         SERIAL PRIMARY KEY,
	build_id   INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	job_id     INT NOT NULL REFERENCES build_jobs(id),
	hash       VARCHAR NOT NULL UNIQUE,
	source     VARCHAR NOT NULL,
	name       VARCHAR NOT NULL,
	size       INT NULL,
	md5        BYTEA NULL,
	sha256     BYTEA NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE build_variables (
	id            SERIAL PRIMARY KEY,
	build_id      INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	variable_id   INT NULL REFERENCES variables(id) ON DELETE SET NULL,
	key           VARCHAR NOT NULL,
	value         VARCHAR NOT NULL
);

CREATE TABLE build_drivers (
	id         SERIAL PRIMARY KEY,
	build_id   INT NOT NULL REFERENCES builds(id) ON DELETE cascade,
	type       driver_type NOT NULL,
	config     JSON NOT NULL
);

CREATE TABLE build_triggers (
	id         SERIAL PRIMARY KEY,
	build_id   INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	type       trigger_type NOT NULL,
	comment    TEXT NOT NULL,
	data       JSON NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE build_keys (
	id         SERIAL PRIMARY KEY,
	build_id   INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	key_id     INT NULL REFERENCES keys(id) ON DELETE SET NULL,
	name       VARCHAR NOT NULL,
	key        BYTEA NOT NULL,
	config     TEXT NOT NULL,
	location   VARCHAR NOT NULL
);

CREATE TABLE providers (
	id               SERIAL PRIMARY KEY,
	user_id          INT NOT NULL REFERENCES users(id),
	provider_user_id INT NULL,
	name             VARCHAR NOT NULL,
	access_token     BYTEA NULL,
	refresh_token    BYTEA NULL,
	connected        BOOLEAN NOT NULL DEFAULT FALSE,
	expires_at       TIMESTAMP NULL
);

CREATE TABLE oauth_apps (
	id            SERIAL PRIMARY KEY,
	user_id       INT NOT NULL REFERENCES users(id),
	client_id     BYTEA NOT NULL UNIQUE,
	client_secret BYTEA NOT NULL,
	name          VARCHAR NOT NULL,
	description   VARCHAR NULL,
	domain        VARCHAR NULL,
	home_uri      VARCHAR NOT NULL,
	redirect_uri  VARCHAR NOT NULL,
	created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE oauth_codes (
	id         SERIAL PRIMARY KEY,
	user_id    INT NOT NULL REFERENCES users(id),
	code       BYTEA NOT NULL UNIQUE,
	scope      BYTEA NOT NULL,
	expires_at TIMESTAMP NOT NULL
);

CREATE TABLE oauth_tokens (
	id         SERIAL PRIMARY KEY,
	user_id    INT NOT NULL REFERENCES users(id),
	app_id     INT NULL REFERENCES oauth_apps(id),
	name       VARCHAR NULL,
	token      BYTEA NOT NULL,
	scope      BYTEA NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
