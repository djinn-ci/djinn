CREATE TABLE build_keys (
	id         SERIAL PRIMARY KEY,
	build_id   INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	key_id     INT NULL REFERENCES keys(id) ON DELETE SET NULL,
	name       VARCHAR NOT NULL,
	key        BYTEA NOT NULL,
	config     TEXT NOT NULL,
	location   VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
