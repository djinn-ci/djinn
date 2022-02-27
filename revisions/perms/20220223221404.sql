/*
Revision: perms/20220223221404
Author:   Andrew Pillar <me@andrewpillar.com>

Grant DELETE permission to djinn_server for webhook deliveries
*/

GRANT DELETE ON namespace_webhook_deliveries TO djinn_server;
