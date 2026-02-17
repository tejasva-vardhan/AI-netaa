-- Authority Dashboard Database Schema
-- PILOT: Minimal authority authentication and notes system

-- Authority authentication credentials (pilot: simple email + password_hash)
-- In production: Use proper password hashing (bcrypt, argon2)
CREATE TABLE IF NOT EXISTS authority_credentials (
    credential_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    officer_id BIGINT NOT NULL COMMENT 'Links to officers table',
    email VARCHAR(255) UNIQUE NOT NULL COMMENT 'Authority email (login)',
    password_hash VARCHAR(255) NOT NULL COMMENT 'Hashed password (pilot: simple hash)',
    static_token VARCHAR(64) NULL COMMENT 'Pilot: Static token for API access',
    token_expires_at TIMESTAMP NULL COMMENT 'Token expiration (if using tokens)',
    is_active BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Account active status',
    last_login_at TIMESTAMP NULL COMMENT 'Last successful login',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    
    FOREIGN KEY (officer_id) REFERENCES officers(officer_id) ON DELETE CASCADE,
    INDEX idx_email (email),
    INDEX idx_officer_id (officer_id),
    INDEX idx_static_token (static_token),
    INDEX idx_is_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Authority notes: Internal notes added by authorities (NOT visible to citizens)
CREATE TABLE IF NOT EXISTS authority_notes (
    note_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    complaint_id BIGINT NOT NULL COMMENT 'Related complaint',
    officer_id BIGINT NOT NULL COMMENT 'Authority who added note',
    note_text TEXT NOT NULL COMMENT 'Internal note content',
    is_visible_to_citizen BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Pilot: Always false (internal only)',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    
    FOREIGN KEY (complaint_id) REFERENCES complaints(complaint_id) ON DELETE CASCADE,
    FOREIGN KEY (officer_id) REFERENCES officers(officer_id) ON DELETE CASCADE,
    INDEX idx_complaint_id (complaint_id),
    INDEX idx_officer_id (officer_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
