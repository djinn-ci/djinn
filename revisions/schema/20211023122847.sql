/*
Revision: schema/20211023122847
Author:   Andrew Pillar <me@andrewpillar.com>

Create webhook tables
*/

CREATE TABLE namespace_webhooks (
	id           SERIAL PRIMARY KEY,
	user_id      INT NOT NULL REFERENCES users(id),
	author_id    INT NOT NULL REFERENCES users(id),
	namespace_id INT NOT NULL REFERENCES namespaces(id),
	payload_url  VARCHAR NOT NULL,
	secret       BYTEA NULL,
	ssl          BOOLEAN NOT NULL,
	events       INT NOT NULL,
	active       BOOLEAN NOT NULL DEFAULT TRUE,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE namespace_webhook_deliveries (
	id               SERIAL,
	webhook_id       INT NOT NULL REFERENCES namespace_webhooks(id) ON DELETE CASCADE,
	event_id         BYTEA NOT NULL,
	error            VARCHAR NULL,
	event            INT NOT NULL,
	redelivery       BOOLEAN NOT NULL DEFAULT FALSE,
	request_headers  VARCHAR NULL,
	request_body     VARCHAR NULL,
	response_code    INT NULL,
	response_headers VARCHAR NULL,
	response_body    VARCHAR NULL,
	duration         INT NOT NULL,
	created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
	PRIMARY KEY(id, event_id)
);
