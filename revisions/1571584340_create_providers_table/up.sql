CREATE TABLE providers (
	id            SERIAL PRIMARY KEY,
	user_id       INT NOT NULL REFERENCES users(id),
	name          VARCHAR NOT NULL,
	access_token  BYTEA NOT NULL,
	refresh_token BYTEA NOT NULL,
	expires_at    TIMESTAMP NOT NULL,
	created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);
