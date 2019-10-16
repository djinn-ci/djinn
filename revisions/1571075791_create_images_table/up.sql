CREATE TABLE images (
	id           SERIAL PRIMARY KEY,
	user_id      INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	namespace_id INT NULL REFERENCES namespaces(id) ON DELETE SET NULL,
	driver       driver_type NOT NULL,
	hash         VARCHAR NOT NULL,
	name         VARCHAR NOT NULL,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);
