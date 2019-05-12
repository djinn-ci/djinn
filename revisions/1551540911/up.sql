CREATE TYPE status AS ENUM ('queued', 'running', 'passed', 'failed', 'passed_with_failures');

CREATE TABLE builds (
	id           SERIAL PRIMARY KEY,
	user_id      INT NOT NULL REFERENCES users(id),
	namespace_id INT NULL REFERENCES namespaces(id) ON DELETE SET NULL,
	manifest     TEXT NOT NULL,
	status       status DEFAULT 'queued',
	output       TEXT NULL,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
	started_at   TIMESTAMP NULL,
	finished_at  TIMESTAMP NULL
);

CREATE TABLE tags (
	id         SERIAL PRIMARY KEY,
	user_id    INT NOT NULL REFERENCES users(id),
	build_id   INT NOT NULL REFERENCES builds(id),
	name       VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
