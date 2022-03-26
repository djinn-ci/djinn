/*
Revision: schema/20220301202019
Author:   Andrew Pillar <me@andrewpillar.com>

Add columns for variable masking
*/

ALTER TABLE variables ADD COLUMN masked BOOLEAN DEFAULT FALSE;
ALTER TABLE build_variables ADD COLUMN masked BOOLEAN DEFAULT FALSE;
