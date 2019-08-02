CREATE TABLE keys (
	id         SERIAL PRIMARY KEY,
	user_id    INT NOT NULL REFERENCES users(id),
	name       VARCHAR NOT NULL,
	key        BYTEA NOT NULL,
	config     TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
