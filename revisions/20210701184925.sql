/*
Revision: 20210701184925
Author:   Andrew Pillar <me@andrewpillar.com>

Create image_downloads table
*/

CREATE TABLE image_downloads (
	id          SERIAL PRIMARY KEY,
	image_id    INT NOT NULL REFERENCES images(id),
	source      VARCHAR NOT NULL,
	error       VARCHAR NULL,
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	started_at  TIMESTAMP NULL,
	finished_at TIMESTAMP NULL
);

GRANT SELECT ON image_downloads TO djinn_server;
GRANT SELECT, INSERT, UPDATE ON image_downloads TO djinn_consumer;
