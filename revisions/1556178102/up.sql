CREATE TABLE artifacts (
	id         SERIAL PRIMARY KEY,
	build_id   INT NOT NULL REFERENCES builds(id),
	job_id     INT NOT NULL REFERENCES jobs(id),
	source     VARCHAR NOT NULL,
	name       VARCHAR NOT NULL,
	size       INT NULL,
	type       VARCHAR NULL,
	md5        BYTEA NULL,
	sha256     BYTEA NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
