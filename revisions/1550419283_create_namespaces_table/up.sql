CREATE TYPE visibility AS ENUM ('private', 'internal', 'public');

CREATE TABLE namespaces (
	id          SERIAL PRIMARY KEY,
	user_id     INT NOT NULL REFERENCES users(id),
	root_id     INT NULL,
	parent_id   INT NULL,
	name        VARCHAR NOT NULL,
	path        VARCHAR NOT NULL,
	description VARCHAR NULL,
	level       INT NOT NULL,
	visibility  visibility DEFAULT 'private',
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
