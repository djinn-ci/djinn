-- mgrt: revision: 1550419283: Create namespaces table
-- mgrt: up

CREATE TYPE visibility AS ENUM ('private', 'internal', 'public');

CREATE TABLE namespaces (
	id          SERIAL PRIMARY KEY,
	user_id     INT NOT NULL REFERENCES users(id),
	parent_id   INT NULL,
	name        VARCHAR(64) NOT NULL,
	full_name   VARCHAR(640) NOT NULL,
	description VARCHAR(255) NULL,
	visibility  visibility DEFAULT 'private',
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

-- mgrt: down

DROP TABLE namespaces;
DROP TYPE visibility;
