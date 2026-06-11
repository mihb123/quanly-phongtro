ALTER TABLE auth_sessions DROP COLUMN IF EXISTS latitude;
ALTER TABLE auth_sessions DROP COLUMN IF EXISTS longitude;
ALTER TABLE auth_sessions DROP COLUMN IF EXISTS geocoding_source;
