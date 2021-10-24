/*
Revision: perms/20211023163649
Author:   Andrew Pillar <me@andrewpillar.com>

Create djinn_consumer user
*/

CREATE USER djinn_consumer;

GRANT CONNECT ON DATABASE djinn TO djinn_consumer;
GRANT SELEC ON image_downloads TO djinn_server;

GRANT SELECT ON images TO djinn_consumer;
GRANT SELECT, UPDATE ON image_downloads TO djinn_consumer;
