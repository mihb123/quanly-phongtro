-- User appearance preferences: color theme + light/dark mode.
-- theme:      existing users keep the historical default (lime); newly created
--             accounts default to sky (the ALTER ... SET DEFAULT below).
-- color_mode: light/dark appearance; defaults to following the OS ('system').
ALTER TABLE users ADD COLUMN IF NOT EXISTS theme VARCHAR(20) NOT NULL DEFAULT 'lime';
ALTER TABLE users ALTER COLUMN theme SET DEFAULT 'sky';
ALTER TABLE users ADD COLUMN IF NOT EXISTS color_mode VARCHAR(20) NOT NULL DEFAULT 'system';
