CREATE TABLE collaborators (
	id           SERIAL PRIMARY KEY,
	namespace_id INT NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
	user_id      INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);
