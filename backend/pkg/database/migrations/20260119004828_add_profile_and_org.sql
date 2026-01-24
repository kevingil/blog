-- +goose Up
-- +goose StatementBegin

-- Organization table
CREATE TABLE IF NOT EXISTS organization (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    bio TEXT,
    logo_url TEXT,
    website_url TEXT,
    email_public VARCHAR(255),
    social_links JSONB DEFAULT '{}',
    meta_description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_organization_slug ON organization(slug);

-- Add profile fields to account + optional organization relationship
ALTER TABLE account
ADD COLUMN IF NOT EXISTS bio TEXT,
ADD COLUMN IF NOT EXISTS profile_image TEXT,
ADD COLUMN IF NOT EXISTS email_public VARCHAR(255),
ADD COLUMN IF NOT EXISTS social_links JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS meta_description TEXT,
ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organization(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_account_organization_id ON account(organization_id);

-- Site settings table (single row)
CREATE TABLE IF NOT EXISTS site_settings (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    public_profile_type VARCHAR(20) DEFAULT 'user' CHECK (public_profile_type IN ('user', 'organization')),
    public_user_id UUID REFERENCES account(id) ON DELETE SET NULL,
    public_organization_id UUID REFERENCES organization(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default row
INSERT INTO site_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS site_settings;

ALTER TABLE account
DROP COLUMN IF EXISTS bio,
DROP COLUMN IF EXISTS profile_image,
DROP COLUMN IF EXISTS email_public,
DROP COLUMN IF EXISTS social_links,
DROP COLUMN IF EXISTS meta_description,
DROP COLUMN IF EXISTS organization_id;

DROP TABLE IF EXISTS organization;
-- +goose StatementEnd
