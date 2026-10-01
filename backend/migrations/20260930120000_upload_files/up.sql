CREATE TABLE upload_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    s3_key TEXT NOT NULL UNIQUE,
    public_url TEXT NOT NULL,
    filename TEXT NOT NULL,
    directory_path TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL,
    byte_size BIGINT NOT NULL,
    width INTEGER,
    height INTEGER,
    blurhash TEXT,
    created_by UUID REFERENCES account(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_upload_files_directory_path ON upload_files(directory_path);
CREATE INDEX idx_upload_files_public_url ON upload_files(public_url);

CREATE TABLE upload_file_refs (
    upload_file_id UUID NOT NULL REFERENCES upload_files(id) ON DELETE CASCADE,
    owner_kind TEXT NOT NULL,
    owner_id UUID NOT NULL,
    PRIMARY KEY (upload_file_id, owner_kind, owner_id)
);

ALTER TABLE article
    ADD COLUMN draft_upload_file_id UUID REFERENCES upload_files(id),
    ADD COLUMN published_upload_file_id UUID REFERENCES upload_files(id);

ALTER TABLE article_version
    ADD COLUMN upload_file_id UUID REFERENCES upload_files(id);

ALTER TABLE page
    ADD COLUMN upload_file_id UUID REFERENCES upload_files(id);

ALTER TABLE project
    ADD COLUMN upload_file_id UUID REFERENCES upload_files(id);

ALTER TABLE account
    ADD COLUMN profile_upload_file_id UUID REFERENCES upload_files(id);

ALTER TABLE organization
    ADD COLUMN logo_upload_file_id UUID REFERENCES upload_files(id);

ALTER TABLE imagen_request
    ADD COLUMN upload_file_id UUID REFERENCES upload_files(id);
