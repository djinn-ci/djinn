/*
Revision: 20210825101049
Author:   Andrew Pillar <me@andrewpillar.com>

Update djinn_worker permissions
*/

GRANT SELECT, INSERT ON build_tags TO djinn_worker;
GRANT USAGE ON SEQUENCE build_tags_id_seq TO djinn_worker;
