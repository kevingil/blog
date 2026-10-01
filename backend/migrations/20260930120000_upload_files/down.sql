ALTER TABLE imagen_request DROP COLUMN IF EXISTS upload_file_id;
ALTER TABLE organization DROP COLUMN IF EXISTS logo_upload_file_id;
ALTER TABLE account DROP COLUMN IF EXISTS profile_upload_file_id;
ALTER TABLE project DROP COLUMN IF EXISTS upload_file_id;
ALTER TABLE page DROP COLUMN IF EXISTS upload_file_id;
ALTER TABLE article_version DROP COLUMN IF EXISTS upload_file_id;
ALTER TABLE article DROP COLUMN IF EXISTS published_upload_file_id;
ALTER TABLE article DROP COLUMN IF EXISTS draft_upload_file_id;

DROP TABLE IF EXISTS upload_file_refs;
DROP TABLE IF EXISTS upload_files;
