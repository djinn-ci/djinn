CREATE TYPE driver_type AS ENUM ('ssh', 'qemu', 'docker');

CREATE TABLE drivers (
	id         SERIAL PRIMARY KEY,
	build_id   INT NOT NULL REFERENCES builds(id) ON DELETE cascade,
	type       driver_type NOT NULL,
	config     JSON NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
