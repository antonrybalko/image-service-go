-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied

-- Create images table
CREATE TABLE IF NOT EXISTS images (
    guid UUID PRIMARY KEY,
    owner_guid UUID NOT NULL,
    type_name TEXT NOT NULL,
    small_url TEXT NOT NULL,
    medium_url TEXT NOT NULL,
    large_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    content_type TEXT,
    original_width INTEGER,
    original_height INTEGER
);

-- Create indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_images_owner_type ON images (owner_guid, type_name);
CREATE INDEX IF NOT EXISTS idx_images_type ON images (type_name);
CREATE INDEX IF NOT EXISTS idx_images_updated_at ON images (updated_at DESC);

-- Add comment to the table
COMMENT ON TABLE images IS 'Stores metadata for images processed by the image service';

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back

-- Drop indexes
DROP INDEX IF EXISTS idx_images_updated_at;
DROP INDEX IF EXISTS idx_images_type;
DROP INDEX IF EXISTS idx_images_owner_type;

-- Drop table
DROP TABLE IF EXISTS images;
