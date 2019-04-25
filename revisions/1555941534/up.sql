CREATE TABLE objects (
	id         SERIAL PRIMARY KEY,
	user_id    INT NOT NULL REFERENCES users(id),
	name       VARCHAR NOT NULL,
	filename   VARCHAR NOT NULL,
	type       VARCHAR NOT NULL,
	size       INT NOT NULL,
	md5        BYTEA NOT NULL,
	sha256     BYTEA NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	deleted_at TIMESTAMP NULL
);

CREATE TABLE build_objects (
	id         SERIAL PRIMARY KEY,
	build_id   INT NOT NULL REFERENCES builds(id),
	object_id  INT NOT NULL REFERENCES objects(id),
	source     VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
