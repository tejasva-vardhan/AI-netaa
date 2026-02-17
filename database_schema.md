# Database Schema Design: Public Accountability System

## Overview
This schema supports a public accountability platform where citizens can file complaints, track their status through an escalation hierarchy, and view transparent timelines. The design emphasizes immutability, auditability, and legal defensibility.

---

## Core Tables

### 1. `users`
**Purpose**: Stores citizen/user information with phone-based verification.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `user_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `phone_number` | VARCHAR(15) | UNIQUE, NOT NULL | Phone number (E.164 format) |
| `phone_verified_at` | TIMESTAMP | NULL | When phone verification completed |
| `phone_verification_code` | VARCHAR(10) | NULL | Temporary OTP (hashed) |
| `phone_verification_expires_at` | TIMESTAMP | NULL | OTP expiration time |
| `full_name` | VARCHAR(255) | NULL | User's full name (optional) |
| `email` | VARCHAR(255) | NULL | Email address (optional) |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Account creation time |
| `last_active_at` | TIMESTAMP | NULL | Last activity timestamp |
| `is_blocked` | BOOLEAN | NOT NULL, DEFAULT FALSE | Account suspension flag |
| `blocked_reason` | TEXT | NULL | Reason for blocking |
| `blocked_at` | TIMESTAMP | NULL | When account was blocked |

**Notes**: Phone number is the primary identifier. Verification is required before filing complaints.

---

### 2. `locations`
**Purpose**: Geographic hierarchy for mapping complaints to departments and officers.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `location_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `location_type` | ENUM('country', 'state', 'district', 'city', 'ward', 'pincode') | NOT NULL | Hierarchy level |
| `name` | VARCHAR(255) | NOT NULL | Location name |
| `code` | VARCHAR(50) | NULL | Official code (e.g., state code, pincode) |
| `parent_location_id` | BIGINT | NULL, FOREIGN KEY → locations.location_id | Parent in hierarchy |
| `latitude` | DECIMAL(10, 8) | NULL | Geographic coordinate |
| `longitude` | DECIMAL(11, 8) | NULL | Geographic coordinate |
| `is_active` | BOOLEAN | NOT NULL, DEFAULT TRUE | Active status |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Record creation |

**Notes**: Self-referencing table for hierarchical location data. Enables geographic routing of complaints.

---

### 3. `departments`
**Purpose**: Government departments/organizations that handle complaints.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `department_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `name` | VARCHAR(255) | NOT NULL | Department name |
| `code` | VARCHAR(50) | UNIQUE, NULL | Official department code |
| `description` | TEXT | NULL | Department description |
| `parent_department_id` | BIGINT | NULL, FOREIGN KEY → departments.department_id | Parent department |
| `is_active` | BOOLEAN | NOT NULL, DEFAULT TRUE | Active status |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Record creation |
| `updated_at` | TIMESTAMP | NULL | Last update |

**Notes**: Supports hierarchical department structures (e.g., state → district → local).

---

### 4. `officers`
**Purpose**: Government officers assigned to handle complaints.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `officer_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `employee_id` | VARCHAR(100) | UNIQUE, NULL | Official employee ID |
| `full_name` | VARCHAR(255) | NOT NULL | Officer's name |
| `designation` | VARCHAR(255) | NULL | Job title/designation |
| `email` | VARCHAR(255) | NULL | Official email |
| `phone_number` | VARCHAR(15) | NULL | Contact number |
| `department_id` | BIGINT | NOT NULL, FOREIGN KEY → departments.department_id | Assigned department |
| `location_id` | BIGINT | NOT NULL, FOREIGN KEY → locations.location_id | Jurisdiction location |
| `is_active` | BOOLEAN | NOT NULL, DEFAULT TRUE | Active status |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Record creation |
| `updated_at` | TIMESTAMP | NULL | Last update |

**Notes**: Links officers to departments and geographic locations for complaint routing.

---

### 5. `department_location_mapping`
**Purpose**: Maps departments to locations (which departments operate in which areas).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `mapping_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `department_id` | BIGINT | NOT NULL, FOREIGN KEY → departments.department_id | Department |
| `location_id` | BIGINT | NOT NULL, FOREIGN KEY → locations.location_id | Location |
| `is_active` | BOOLEAN | NOT NULL, DEFAULT TRUE | Active status |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Record creation |
| `updated_at` | TIMESTAMP | NULL | Last update |

**Unique Constraint**: `(department_id, location_id)`

**Notes**: Many-to-many relationship. Determines which departments handle complaints in specific locations.

---

### 6. `complaints`
**Purpose**: Main complaint records filed by citizens.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `complaint_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `complaint_number` | VARCHAR(50) | UNIQUE, NOT NULL | Public-facing complaint number |
| `user_id` | BIGINT | NOT NULL, FOREIGN KEY → users.user_id | Complainant |
| `title` | VARCHAR(500) | NOT NULL | Complaint title |
| `description` | TEXT | NOT NULL | Detailed description |
| `category` | VARCHAR(100) | NULL | Complaint category |
| `location_id` | BIGINT | NOT NULL, FOREIGN KEY → locations.location_id | Complaint location |
| `latitude` | DECIMAL(10, 8) | NULL | Specific coordinates |
| `longitude` | DECIMAL(11, 8) | NULL | Specific coordinates |
| `assigned_department_id` | BIGINT | NULL, FOREIGN KEY → departments.department_id | Initially assigned department |
| `assigned_officer_id` | BIGINT | NULL, FOREIGN KEY → officers.officer_id | Currently assigned officer |
| `current_status` | ENUM('draft', 'submitted', 'under_review', 'in_progress', 'resolved', 'rejected', 'closed', 'escalated') | NOT NULL, DEFAULT 'draft' | Current status |
| `priority` | ENUM('low', 'medium', 'high', 'urgent') | NOT NULL, DEFAULT 'medium' | Priority level |
| `is_public` | BOOLEAN | NOT NULL, DEFAULT FALSE | Public visibility flag |
| `public_consent_given` | BOOLEAN | NOT NULL, DEFAULT FALSE | User consent for public disclosure |
| `supporter_count` | INT | NOT NULL, DEFAULT 0 | Count of supporting users |
| `resolved_at` | TIMESTAMP | NULL | Resolution timestamp |
| `closed_at` | TIMESTAMP | NULL | Closure timestamp |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Complaint creation |
| `updated_at` | TIMESTAMP | NULL | Last update (for non-audit changes) |

**Notes**: Core complaint entity. Status changes are tracked via `complaint_status_history`. `supporter_count` is denormalized for performance.

---

### 7. `complaint_status_history`
**Purpose**: Immutable timeline of complaint status changes (append-only).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `history_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `complaint_id` | BIGINT | NOT NULL, FOREIGN KEY → complaints.complaint_id | Related complaint |
| `old_status` | ENUM('draft', 'submitted', 'under_review', 'in_progress', 'resolved', 'rejected', 'closed', 'escalated') | NULL | Previous status |
| `new_status` | ENUM('draft', 'submitted', 'under_review', 'in_progress', 'resolved', 'rejected', 'closed', 'escalated') | NOT NULL | New status |
| `changed_by_type` | ENUM('user', 'officer', 'system', 'admin') | NOT NULL | Who made the change |
| `changed_by_user_id` | BIGINT | NULL, FOREIGN KEY → users.user_id | User who changed (if applicable) |
| `changed_by_officer_id` | BIGINT | NULL, FOREIGN KEY → officers.officer_id | Officer who changed (if applicable) |
| `assigned_department_id` | BIGINT | NULL, FOREIGN KEY → departments.department_id | Department at time of change |
| `assigned_officer_id` | BIGINT | NULL, FOREIGN KEY → officers.officer_id | Officer at time of change |
| `notes` | TEXT | NULL | Status change notes/comments |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Change timestamp |

**Index**: `(complaint_id, created_at DESC)` for timeline queries

**Notes**: Append-only table. Every status change creates a new record. Provides complete audit trail visible to citizens.

---

### 8. `complaint_escalations`
**Purpose**: Tracks escalation hierarchy and escalation events.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `escalation_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `complaint_id` | BIGINT | NOT NULL, FOREIGN KEY → complaints.complaint_id | Escalated complaint |
| `from_department_id` | BIGINT | NULL, FOREIGN KEY → departments.department_id | Department escalated from |
| `from_officer_id` | BIGINT | NULL, FOREIGN KEY → officers.officer_id | Officer escalated from |
| `to_department_id` | BIGINT | NOT NULL, FOREIGN KEY → departments.department_id | Department escalated to |
| `to_officer_id` | BIGINT | NULL, FOREIGN KEY → officers.officer_id | Officer escalated to |
| `escalation_level` | INT | NOT NULL | Level in hierarchy (1, 2, 3...) |
| `reason` | TEXT | NULL | Escalation reason |
| `escalated_by_type` | ENUM('user', 'officer', 'system', 'admin') | NOT NULL | Who escalated |
| `escalated_by_user_id` | BIGINT | NULL, FOREIGN KEY → users.user_id | User who escalated (if applicable) |
| `escalated_by_officer_id` | BIGINT | NULL, FOREIGN KEY → officers.officer_id | Officer who escalated (if applicable) |
| `status_history_id` | BIGINT | NULL, FOREIGN KEY → complaint_status_history.history_id | Related status change |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Escalation timestamp |

**Index**: `(complaint_id, escalation_level)` for hierarchy queries

**Notes**: Records each escalation step. Links to status history for complete audit trail.

---

### 9. `complaint_supporters`
**Purpose**: Tracks users who support/duplicate a complaint.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `support_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `complaint_id` | BIGINT | NOT NULL, FOREIGN KEY → complaints.complaint_id | Supported complaint |
| `user_id` | BIGINT | NOT NULL, FOREIGN KEY → users.user_id | Supporting user |
| `is_duplicate` | BOOLEAN | NOT NULL, DEFAULT FALSE | Marks as duplicate complaint |
| `duplicate_notes` | TEXT | NULL | Notes if marking as duplicate |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Support timestamp |

**Unique Constraint**: `(complaint_id, user_id)` - one support per user per complaint

**Index**: `(complaint_id)` for counting supporters

**Notes**: Prevents duplicate support. `is_duplicate` flag allows marking similar complaints. `supporter_count` in `complaints` is maintained via triggers or application logic.

---

### 10. `user_consents`
**Purpose**: Manages user consent for public disclosure of complaints.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `consent_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `user_id` | BIGINT | NOT NULL, FOREIGN KEY → users.user_id | User |
| `complaint_id` | BIGINT | NULL, FOREIGN KEY → complaints.complaint_id | Specific complaint (NULL = global) |
| `consent_type` | ENUM('public_disclosure', 'data_sharing', 'notifications') | NOT NULL | Type of consent |
| `consent_given` | BOOLEAN | NOT NULL | Consent status |
| `consent_text` | TEXT | NULL | Text of consent agreement |
| `ip_address` | VARCHAR(45) | NULL | IP at consent time |
| `user_agent` | VARCHAR(500) | NULL | Browser/device info |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Consent timestamp |
| `revoked_at` | TIMESTAMP | NULL | Revocation timestamp |

**Index**: `(user_id, complaint_id, consent_type)` for consent lookups

**Notes**: Immutable consent records. Revocation creates new record with `consent_given = FALSE`. Supports both per-complaint and global consent.

---

### 11. `audit_log`
**Purpose**: Immutable append-only audit trail for all critical actions.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `audit_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `entity_type` | VARCHAR(100) | NOT NULL | Entity type (e.g., 'complaint', 'user', 'officer') |
| `entity_id` | BIGINT | NOT NULL | Entity identifier |
| `action` | VARCHAR(100) | NOT NULL | Action performed (e.g., 'create', 'update', 'delete', 'status_change') |
| `action_by_type` | ENUM('user', 'officer', 'system', 'admin') | NOT NULL | Actor type |
| `action_by_user_id` | BIGINT | NULL, FOREIGN KEY → users.user_id | User actor |
| `action_by_officer_id` | BIGINT | NULL, FOREIGN KEY → officers.officer_id | Officer actor |
| `old_values` | JSON | NULL | Previous state (JSON snapshot) |
| `new_values` | JSON | NULL | New state (JSON snapshot) |
| `changes` | JSON | NULL | Diff of changes |
| `ip_address` | VARCHAR(45) | NULL | IP address |
| `user_agent` | VARCHAR(500) | NULL | Browser/device info |
| `metadata` | JSON | NULL | Additional context |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Audit timestamp |

**Indexes**: 
- `(entity_type, entity_id, created_at DESC)` for entity history
- `(action_by_type, action_by_user_id, created_at DESC)` for user activity
- `(action_by_type, action_by_officer_id, created_at DESC)` for officer activity

**Notes**: Comprehensive audit trail. JSON fields store flexible data. Never updated or deleted.

---

### 12. `complaint_attachments`
**Purpose**: Stores file attachments (photos, documents) for complaints.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `attachment_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `complaint_id` | BIGINT | NOT NULL, FOREIGN KEY → complaints.complaint_id | Related complaint |
| `file_name` | VARCHAR(255) | NOT NULL | Original filename |
| `file_path` | VARCHAR(1000) | NOT NULL | Storage path/URL |
| `file_type` | VARCHAR(100) | NULL | MIME type |
| `file_size` | BIGINT | NULL | Size in bytes |
| `uploaded_by_user_id` | BIGINT | NULL, FOREIGN KEY → users.user_id | Uploader |
| `is_public` | BOOLEAN | NOT NULL, DEFAULT FALSE | Public visibility |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Upload timestamp |

**Index**: `(complaint_id)` for complaint attachments

**Notes**: Supports evidence/documentation. Respects public visibility consent.

---

### 13. `escalation_rules`
**Purpose**: Defines escalation hierarchy and rules (which department/officer escalates to which).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `rule_id` | BIGINT | PRIMARY KEY, AUTO_INCREMENT | Unique identifier |
| `from_department_id` | BIGINT | NULL, FOREIGN KEY → departments.department_id | Source department (NULL = any) |
| `from_location_id` | BIGINT | NULL, FOREIGN KEY → locations.location_id | Source location (NULL = any) |
| `to_department_id` | BIGINT | NOT NULL, FOREIGN KEY → departments.department_id | Target department |
| `to_location_id` | BIGINT | NULL, FOREIGN KEY → locations.location_id | Target location (NULL = same) |
| `escalation_level` | INT | NOT NULL | Level in hierarchy |
| `conditions` | JSON | NULL | Conditions for escalation (time-based, status-based, etc.) |
| `is_active` | BOOLEAN | NOT NULL, DEFAULT TRUE | Active status |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Rule creation |
| `updated_at` | TIMESTAMP | NULL | Last update |

**Notes**: Configurable escalation matrix. JSON `conditions` allows flexible rules (e.g., auto-escalate after 7 days).

---

## Relationships Summary

1. **Users → Complaints**: One-to-many (users file multiple complaints)
2. **Complaints → Locations**: Many-to-one (each complaint has one location)
3. **Complaints → Departments**: Many-to-one (assigned department)
4. **Complaints → Officers**: Many-to-one (assigned officer)
5. **Departments ↔ Locations**: Many-to-many via `department_location_mapping`
6. **Officers → Departments**: Many-to-one (officer belongs to one department)
7. **Officers → Locations**: Many-to-one (officer jurisdiction)
8. **Complaints → Status History**: One-to-many (immutable timeline)
9. **Complaints → Escalations**: One-to-many (multiple escalation steps)
10. **Complaints → Supporters**: One-to-many (multiple users can support)
11. **Users → Consents**: One-to-many (multiple consent records)
12. **Complaints → Attachments**: One-to-many (multiple files)

---

## Key Design Principles

1. **Immutability**: `complaint_status_history`, `audit_log`, and `user_consents` are append-only
2. **Auditability**: Every critical action is logged in `audit_log` with full context
3. **Scalability**: Indexes on foreign keys and frequently queried fields
4. **Legal Defensibility**: Complete audit trails, consent records, and immutable history
5. **Transparency**: Status timeline visible to citizens via `complaint_status_history`
6. **Geographic Routing**: Location hierarchy enables automatic department/officer assignment
7. **Escalation Support**: Explicit escalation tracking and configurable rules

---

## Additional Considerations

- **Partitioning**: Consider partitioning `audit_log` and `complaint_status_history` by date for large-scale deployments
- **Archiving**: Old resolved complaints can be archived to separate tables while maintaining referential integrity
- **Full-text Search**: Add full-text indexes on `complaints.title` and `complaints.description` for search functionality
- **Soft Deletes**: Consider adding `deleted_at` timestamp fields where logical deletion is needed (not applicable to audit tables)
- **Concurrency**: Use optimistic locking (version fields) or database-level locking for critical updates
- **Data Retention**: Define policies for retention of audit logs and historical data per legal requirements
