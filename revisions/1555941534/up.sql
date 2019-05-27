CREATE TABLE objects (
	id         SERIAL PRIMARY KEY,
	user_id    INT NOT NULL REFERENCES users(id),
	hash       VARCHAR NOT NULL UNIQUE,
	name       VARCHAR NOT NULL,
	type       VARCHAR NOT NULL,
	size       INT NOT NULL,
	md5        BYTEA NOT NULL,
	sha256     BYTEA NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	deleted_at TIMESTAMP NULL
);

CREATE TABLE build_objects (
	id          SERIAL PRIMARY KEY,
	build_id    INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	object_id   INT NOT NULL REFERENCES objects(id),
	source      VARCHAR NOT NULL,
	name        VARCHAR NOT NULL,
	placed      BOOLEAN NOT NULL DEFAULT FALSE,
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
