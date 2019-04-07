CREATE TABLE jobs (
	id          SERIAL PRIMARY KEY,
	stage_id    INT NOT NULL REFERENCES stages(id),
	name        VARCHAR(32) NOT NULL,
	output      TEXT NULL,
	status      status DEFAULT 'queued',
	started_at  TIMESTAMP NULL,
	finished_at TIMESTAMP NULL
);
