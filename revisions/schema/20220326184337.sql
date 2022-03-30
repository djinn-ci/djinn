/*
Revision: schema/20220326184337
Author:   Andrew Pillar <me@andrewpillar.com>

Use BIGINT for sizes
*/


ALTER TABLE objects ALTER COLUMN size TYPE BIGINT USING size::BIGINT;
ALTER TABLE build_artifacts ALTER COLUMN size TYPE BIGINT USING size::BIGINT;
