-- mgrt: revision: 1550419283: Create namespaces table
-- mgrt: up

CREATE TABLE namespaces (
	id          SERIAL PRIMARY KEY,
	user_id     INT NOT NULL REFERENCES users(id),
	name        VARCHAR(64) NOT NULL,
	description VARCHAR(255) NULL,
	private     BOOLEAN NOT NULL DEFAULT TRUE,
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

-- mgrt: down

DROP TABLE namespaces;
