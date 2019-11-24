CREATE TABLE providers (
	id               SERIAL PRIMARY KEY,
	user_id          INT NOT NULL REFERENCES users(id),
	provider_user_id INT NULL,
	name             VARCHAR NOT NULL,
	access_token     BYTEA NULL,
	refresh_token    BYTEA NULL,
	connected        BOOLEAN NOT NULL DEFAULT FALSE,
	expires_at       TIMESTAMP NULL,
	created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at       TIMESTAMP NOT NULL DEFAULT NOW()
);
