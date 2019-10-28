CREATE TABLE repos (
	id          SERIAL PRIMARY KEY,
	user_id     INT NOT NULL REFERENCES users(id),
	provider_id INT NOT NULL REFERENCES providers(id),
	repo_id     INT NOT NULL,
	name        VARCHAR NOT NULL,
	href        VARCHAR NOT NULL,
	enabled     BOOLEAN NOT NULL DEFAULT TRUE,
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
