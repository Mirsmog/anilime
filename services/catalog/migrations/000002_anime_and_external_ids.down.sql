DROP TABLE IF EXISTS external_episode_ids;
ALTER TABLE episodes DROP COLUMN IF EXISTS updated_at;
ALTER TABLE episodes DROP COLUMN IF EXISTS url;
DROP TABLE IF EXISTS external_anime_ids;
DROP TABLE IF EXISTS anime;
