/*
Revision: schema/20221002113314
Author:   Andrew Pillar <me@andrewpillar.com>

Add pinned column to builds
*/

ALTER TABLE builds ADD COLUMN pinned BOOLEAN DEFAULT FALSE;
