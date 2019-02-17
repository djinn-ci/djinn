-- mgrt: revision: 1550356067: Create users table
-- mgrt: up

CREATE TABLE users (
	id         SERIAL PRIMARY KEY,
	email      CHAR(254) NOT NULL UNIQUE,
	username   CHAR(32) NOT NULL UNIQUE,
	password   BYTEA NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	deleted_at TIMESTAMP NULL
);

-- mgrt: down

DROP TABLE users;
