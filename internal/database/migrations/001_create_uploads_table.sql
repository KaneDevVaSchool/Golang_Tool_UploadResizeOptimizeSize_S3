-- Create uploads table for tracking file uploads
CREATE TABLE
    IF NOT EXISTS uploads (
        id BIGSERIAL PRIMARY KEY,
        filename VARCHAR(255) NOT NULL,
        original_name VARCHAR(255) NOT NULL,
        file_size BIGINT NOT NULL,
        content_type VARCHAR(100) NOT NULL,
        s3_key VARCHAR(500),
        s3_url VARCHAR(1000),
        status VARCHAR(20) NOT NULL DEFAULT 'pending',
        error TEXT,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

-- Create index on status for faster queries
CREATE INDEX IF NOT EXISTS idx_uploads_status ON uploads (status);

-- Create index on created_at for sorting
CREATE INDEX IF NOT EXISTS idx_uploads_created_at ON uploads (created_at DESC);

-- Create index on s3_key for lookups
CREATE INDEX IF NOT EXISTS idx_uploads_s3_key ON uploads (s3_key);