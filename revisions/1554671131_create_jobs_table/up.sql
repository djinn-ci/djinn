CREATE TABLE jobs (
	id          SERIAL PRIMARY KEY,
	build_id    INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	stage_id    INT NOT NULL REFERENCES stages(id) ON DELETE CASCADE,
	name        VARCHAR NOT NULL,
	commands    VARCHAR NOT NULL,
	output      TEXT NULL,
	status      status DEFAULT 'queued',
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	started_at  TIMESTAMP NULL,
	finished_at TIMESTAMP NULL
);

CREATE TABLE job_dependencies (
	id            SERIAL PRIMARY KEY,
	job_id        INT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
	dependency_id INT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE
);
