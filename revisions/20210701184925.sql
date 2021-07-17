/*
Revision: 20210701184925
Author:   Andrew Pillar <me@andrewpillar.com>

Create image_downloads table
*/

CREATE TABLE image_downloads (
	id           SERIAL PRIMARY KEY,
	image_id     INT NOT NULL REFERENCES images(id),
	url          VARCHAR NOT NULL,
	error        VARCHAR NULL,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
	started_at   TIMESTAMP NULL,
	finished_at  TIMESTAMP NULL
);