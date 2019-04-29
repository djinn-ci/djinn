CREATE TABLE stages (
	id          SERIAL PRIMARY KEY,
	build_id    INT NOT NULL REFERENCES builds(id),
	name        VARCHAR(32) NOT NULL,
	can_fail    BOOLEAN NOT NULL,
	did_fail    BOOLEAN NOT NULL DEFAULT false,
	status      status DEFAULT 'queued',
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	started_at  TIMESTAMP NULL,
	finished_at TIMESTAMP NULL
);
