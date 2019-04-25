CREATE TABLE jobs (
	id          SERIAL PRIMARY KEY,
	build_id    INT NOT NULL REFERENCES builds(id),
	stage_id    INT NOT NULL REFERENCES stages(id),
	parent      INT NULL,
	name        VARCHAR(32) NOT NULL,
	commands    VARCHAR NOT NULL,
	output      TEXT NULL,
	status      status DEFAULT 'queued',
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	started_at  TIMESTAMP NULL,
	finished_at TIMESTAMP NULL
);
