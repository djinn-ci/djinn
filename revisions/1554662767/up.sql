CREATE TABLE stages (
	id          SERIAL PRIMARY KEY,
	build_id    INT NOT NULL REFERENCES builds(id),
	name        VARCHAR(32) NOT NULL,
	started_at  TIMESTAMP NULL,
	finished_at TIMESTAMP NULL
);
