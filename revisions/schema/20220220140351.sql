/*
Revision: schema/20220220140351
Author:   Andrew Pillar <me@andrewpillar.com>

Remove unused build_stages columns
*/

ALTER TABLE build_stages DROP COLUMN status,
	DROP COLUMN started_at,
	DROP COLUMN finished_at;
