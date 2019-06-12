CREATE TYPE trigger_type AS ENUM ('manual');

CREATE TABLE triggers (
	id         SERIAL PRIMARY KEY,
	build_id   INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	type       trigger_type NOT NULL,
	comment    TEXT NOT NULL,
	data       JSON NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
