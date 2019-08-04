CREATE TABLE variables (
	id            SERIAL PRIMARY KEY,
	user_id       INT NOT NULL REFERENCES users(id),
	namespace_id  INT NULL REFERENCES namespaces(id) ON DELETE SET NULL,
	key           VARCHAR NOT NULL,
	value         VARCHAR NOT NULL,
	created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE build_variables (
	id            SERIAL PRIMARY KEY,
	build_id      INT NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
	variable_id   INT NULL REFERENCES variables(id),
	key           VARCHAR NOT NULL,
	value         VARCHAR NOT NULL,
	created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);
