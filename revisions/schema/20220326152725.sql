/*
Revision: schema/20220326152725
Author:   Andrew Pillar <me@andrewpillar.com>

Change cleanup to be a number
*/

ALTER TABLE users
	ALTER COLUMN cleanup DROP DEFAULT,
	ALTER COLUMN cleanup TYPE INT USING cleanup::INT,
	ALTER COLUMN cleanup SET DEFAULT 1073741824;

ALTER TABLE users ALTER COLUMN cleanup TYPE BIGINT USING cleanup::BIGINT;

UPDATE users SET cleanup = 1073741824;
