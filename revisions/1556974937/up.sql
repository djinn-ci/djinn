CREATE TABLE variables (
	id            SERIAL PRIMARY KEY,
	user_id       INT NOT NULL REFERENCES users(id),
	key           VARCHAR NOT NULL,
	value         VARCHAR NOT NULL,
	from_manifest BOOLEAN NOT NULL DEFAULT FALSE,
	created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at    TIMESTAMP NOT NULL DEFAULT NOW(),
	deleted_at    TIMESTAMP NULL
);

CREATE TABLE build_variables (
	id          SERIAL PRIMARY KEY,
	build_id    INT NOT NULL REFERENCES builds(id),
	variable_id INT NOT NULL REFERENCES variables(id),
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
