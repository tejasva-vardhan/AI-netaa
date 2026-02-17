-- Authority pilot: login and notes only. officers in core schema; department_id/location_id are valid identifiers.

CREATE TABLE IF NOT EXISTS officers (
    officer_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    employee_id VARCHAR(100) UNIQUE NULL,
    full_name VARCHAR(255) NOT NULL,
    designation VARCHAR(255) NULL,
    email VARCHAR(255) NULL,
    department_id BIGINT NOT NULL,
    location_id BIGINT NOT NULL,
    authority_level TINYINT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    INDEX idx_department_location (department_id, location_id),
    INDEX idx_is_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS authority_credentials (
    credential_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    officer_id BIGINT NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    static_token VARCHAR(64) NULL,
    token_expires_at TIMESTAMP NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    FOREIGN KEY (officer_id) REFERENCES officers(officer_id) ON DELETE CASCADE,
    INDEX idx_email (email),
    INDEX idx_officer_id (officer_id),
    INDEX idx_static_token (static_token)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS authority_notes (
    note_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    complaint_id BIGINT NOT NULL,
    officer_id BIGINT NOT NULL,
    note_text TEXT NOT NULL,
    is_visible_to_citizen BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    FOREIGN KEY (complaint_id) REFERENCES complaints(complaint_id) ON DELETE CASCADE,
    FOREIGN KEY (officer_id) REFERENCES officers(officer_id) ON DELETE CASCADE,
    INDEX idx_complaint_id (complaint_id),
    INDEX idx_officer_id (officer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
