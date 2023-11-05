/*
Revision: perms/20231105130729
Author:   Andrew Pillar <me@andrewpillar.com>

Webhook permissions
*/

GRANT SELECT ON namespace_webhooks TO djinn_consumer, djinn_worker;
