-- Abuse Prevention Database Schema Changes
-- Adds device fingerprinting and pincode support for complaint abuse prevention

-- Add pincode column to complaints table (if not exists)
ALTER TABLE complaints 
ADD COLUMN IF NOT EXISTS pincode VARCHAR(10) NULL 
COMMENT 'Postal code / PIN code for complaint location';

-- Add device_fingerprint column to complaints table
ALTER TABLE complaints 
ADD COLUMN IF NOT EXISTS device_fingerprint VARCHAR(64) NULL 
COMMENT 'SHA256 hash of user_id + user_agent + screen_size';

-- Indexes for performance (abuse prevention queries)
-- Index for rate limiting: user_id + created_at + current_status
CREATE INDEX IF NOT EXISTS idx_user_created_status 
ON complaints(user_id, created_at DESC, current_status);

-- Index for duplicate detection: user_id + title + pincode + created_at
CREATE INDEX IF NOT EXISTS idx_user_title_pincode_created 
ON complaints(user_id, title(100), pincode, created_at DESC);

-- Index for device fingerprint analysis (optional, for pattern detection)
CREATE INDEX IF NOT EXISTS idx_device_fingerprint 
ON complaints(device_fingerprint, created_at DESC);
