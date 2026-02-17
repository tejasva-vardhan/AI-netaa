-- Voice notes: one per complaint, citizen upload only. Authority can access. Not public.

CREATE TABLE IF NOT EXISTS complaint_voice_notes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    complaint_id BIGINT NOT NULL,
    file_path VARCHAR(255) NOT NULL,
    mime_type VARCHAR(50) NOT NULL,
    duration_seconds INT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (complaint_id) REFERENCES complaints(complaint_id) ON DELETE CASCADE,
    UNIQUE KEY uk_complaint_voice (complaint_id),
    INDEX idx_complaint_id (complaint_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
