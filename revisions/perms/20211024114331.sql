/*
Revision: perms/20211024114331
Author:   Andrew Pillar <me@andrewpillar.com>

Grant webhook permissions
*/

GRANT SELECT, INSERT ON namespace_webhook_deliveries TO djinn_consumer, djinn_worker;
GRANT USAGE ON SEQUENCE namespace_webhook_deliveries_id_seq TO djinn_consumer, djinn_worker;
