CREATE TABLE oauth_apps (
	id            SERIAL PRIMARY KEY,
	user_id       INT NOT NULL REFERENCES users(id),
	client_id     BYTEA NOT NULL UNIQUE,
	client_secret BYTEA NOT NULL,
	name          VARCHAR NOT NULL,
	description   VARCHAR NULL,
	domain        VARCHAR NULL,
	home_uri      VARCHAR NOT NULL,
	redirect_uri  VARCHAR NOT NULL,
	created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE oauth_codes (
	id         SERIAL PRIMARY KEY,
	user_id    INT NOT NULL REFERENCES users(id),
	code       BYTEA NOT NULL UNIQUE,
	scope      BYTEA NOT NULL,
	expires_at TIMESTAMP NOT NULL
);

CREATE TABLE oauth_tokens (
	id         SERIAL PRIMARY KEY,
	user_id    INT NOT NULL REFERENCES users(id),
	app_id     INT NULL REFERENCES oauth_apps(id),
	name       VARCHAR NULL,
	token      BYTEA NOT NULL,
	scope      BYTEA NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
