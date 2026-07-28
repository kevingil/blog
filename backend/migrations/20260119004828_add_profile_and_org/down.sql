DROP TABLE IF EXISTS site_settings;

ALTER TABLE account
DROP COLUMN IF EXISTS bio,
DROP COLUMN IF EXISTS profile_image,
DROP COLUMN IF EXISTS email_public,
DROP COLUMN IF EXISTS social_links,
DROP COLUMN IF EXISTS meta_description,
DROP COLUMN IF EXISTS organization_id;

DROP TABLE IF EXISTS organization;
