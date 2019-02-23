-- mgrt: revision: 1550934313: Add visibility to namespaces
-- mgrt: up

CREATE TYPE visibility AS ENUM ('private', 'internal', 'public');

ALTER TABLE namespaces ADD COLUMN visibility visibility;
ALTER TABLE namespaces DROP COLUMN private;

-- mgrt: down

DROP TYPE visibility;
ALTER TABLE namespaces DROP COLUMN visibility;
ALTER TABLE namespaces ADD COLUMN private BOOLEAN DEFAULT TRUE;
