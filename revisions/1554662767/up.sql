CREATE TABLE stages (
	id          SERIAL PRIMARY KEY,
	build_id    INT NOT NULL REFERENCES builds(id),
	name        VARCHAR(32) NOT NULL,
	can_fail    BOOLEAN NOT NULL,
	did_fail    BOOLEAN NOT NULL,
	status      status DEFAULT 'queued',
	started_at  TIMESTAMP NULL,
	finished_at TIMESTAMP NULL
);
