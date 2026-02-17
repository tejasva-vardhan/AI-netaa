-- Evidence Integrity Verification Schema
-- Stores cryptographic hashes for complaint media (integrity signal only; see trust model in docs).
-- Table is WRITE-ONCE: no updates to evidence_hash, latitude, longitude, captured_at after insert.

-- Evidence integrity table: Links to attachments and stores hash + metadata
CREATE TABLE IF NOT EXISTS complaint_evidence (
    evidence_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    attachment_id BIGINT NOT NULL COMMENT 'Related attachment (photo)',
    complaint_id BIGINT NOT NULL COMMENT 'Related complaint',
    evidence_hash VARCHAR(64) NOT NULL COMMENT 'SHA256 of raw image_bytes + lat + lng + server captured_at',
    captured_at TIMESTAMP NOT NULL COMMENT 'Server-side timestamp at upload (do not use client time)',
    latitude DECIMAL(10, 8) NULL COMMENT 'GPS latitude at capture time',
    longitude DECIMAL(11, 8) NULL COMMENT 'GPS longitude at capture time',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Record creation',
    
    FOREIGN KEY (attachment_id) REFERENCES complaint_attachments(attachment_id) ON DELETE CASCADE,
    FOREIGN KEY (complaint_id) REFERENCES complaints(complaint_id) ON DELETE CASCADE,
    UNIQUE KEY uk_attachment_evidence (attachment_id),
    INDEX idx_complaint_id (complaint_id),
    INDEX idx_evidence_hash (evidence_hash),
    INDEX idx_captured_at (captured_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
