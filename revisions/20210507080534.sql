/*
Revision: 20210507080534
Author:   Andrew Pillar <me@andrewpillar.com>

Create webhook tables
*/

CREATE TABLE namespace_webhooks (
	id           SERIAL PRIMARY KEY,
	user_id      INT NOT NULL REFERENCES users(id),
	author_id    INT NOT NULL REFERENCES users(id),
	namespace_id INT NOT NULL REFERENCES namespaces(id),
	payload_url  VARCHAR NOT NULL,
	secret       VARCHAR NULL,
	ssl          BOOLEAN NOT NULL,
	events       BYTEA NOT NULL,
	active       BOOLEAN NOT NULL DEFAULT TRUE,
	created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE namespace_webhook_deliveries (
	id               SERIAL,
	webhook_id       INT NOT NULL REFERENCES namespace_webhooks(id) ON DELETE CASCADE,
	delivery_id      VARCHAR NOT NULL,
	redelivery       BOOLEAN NOT NULL DEFAULT FALSE,
	request_headers  VARCHAR NOT NULL,
	request_body     VARCHAR NOT NULL, 
	response_code    INT NOT NULL,
	repsonse_headers VARCHAR NOT NULL,
	response_body    VARCHAR NULL,
	duration         INT NOT NULL,
	created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
	PRIMARY KEY(id, delivery_id)
);