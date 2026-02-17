# Minimal database schema for Authority (AI Neta)

- **Scope:** Supports Authority authentication, levels (L1/L2/L3), and mapping to department + geography.
- **Constraints:** No changes to existing complaint, escalation, or audit tables. Reuses `complaint_status_history` and `audit_log`. No role-permission framework, no polymorphic design.

---

## 1. officers

- **Why:** One row per Authority (government officer). Links authentication, department, and geography. Referenced by complaints (`assigned_officer_id`), escalation (authority lookup by department + location + level), and audit (actor_id = officer_id).
- **Primary key:** `officer_id` (BIGINT, AUTO_INCREMENT)
- **Columns:**
  - `officer_id` BIGINT PRIMARY KEY AUTO_INCREMENT
  - `employee_id` VARCHAR(100) UNIQUE NULL — official ID; level encoded (e.g. `PWD-L1-001`, `PHED-L2-001`) for L1/L2/L3
  - `full_name` VARCHAR(255) NOT NULL
  - `designation` VARCHAR(255) NULL
  - `email` VARCHAR(255) NULL
  - `phone_number` VARCHAR(15) NULL
  - `department_id` BIGINT NOT NULL — FK to departments
  - `location_id` BIGINT NOT NULL — FK to locations (geography)
  - `is_active` BOOLEAN NOT NULL DEFAULT TRUE
  - `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
  - `updated_at` TIMESTAMP NULL
- **Indexes:** `department_id`, `location_id`, `(department_id, location_id)`, `is_active`, `employee_id`
- **Foreign keys:** None in this table (departments/locations may be created after officers in bootstrap; or FK to `departments(department_id)`, `locations(location_id)` if both exist first)

---

## 2. authority_credentials

- **Why:** Authority login. One row per officer with email/password (and optional pilot static token). Used only for authentication; no complaint or escalation data.
- **Primary key:** `credential_id` (BIGINT, AUTO_INCREMENT)
- **Columns:**
  - `credential_id` BIGINT PRIMARY KEY AUTO_INCREMENT
  - `officer_id` BIGINT NOT NULL — which officer can log in
  - `email` VARCHAR(255) UNIQUE NOT NULL — login identifier
  - `password_hash` VARCHAR(255) NOT NULL
  - `static_token` VARCHAR(64) NULL — pilot: static API token
  - `token_expires_at` TIMESTAMP NULL
  - `is_active` BOOLEAN NOT NULL DEFAULT TRUE
  - `last_login_at` TIMESTAMP NULL
  - `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
  - `updated_at` TIMESTAMP NULL
- **Foreign key:** `officer_id` → `officers(officer_id)` ON DELETE CASCADE
- **Indexes:** `email`, `officer_id`, `static_token`, `is_active`

---

## 3. authority_notes

- **Why:** Internal notes added by an Authority on a complaint. Not visible to citizens. Audit uses existing `audit_log` (action = add_note, action_by_officer_id).
- **Primary key:** `note_id` (BIGINT, AUTO_INCREMENT)
- **Columns:**
  - `note_id` BIGINT PRIMARY KEY AUTO_INCREMENT
  - `complaint_id` BIGINT NOT NULL
  - `officer_id` BIGINT NOT NULL — Authority who wrote the note
  - `note_text` TEXT NOT NULL
  - `is_visible_to_citizen` BOOLEAN NOT NULL DEFAULT FALSE — pilot: always false
  - `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
  - `updated_at` TIMESTAMP NULL
- **Foreign keys:** `complaint_id` → `complaints(complaint_id)` ON DELETE CASCADE; `officer_id` → `officers(officer_id)` ON DELETE CASCADE
- **Indexes:** `complaint_id`, `officer_id`, `created_at`

---

## 4. departments

- **Why:** Authority is mapped to department + geography. Routing and escalation use `department_id`. Officers and complaints reference it; no department table means no referential clarity or seed data.
- **Primary key:** `department_id` (BIGINT, AUTO_INCREMENT)
- **Columns:**
  - `department_id` BIGINT PRIMARY KEY AUTO_INCREMENT
  - `name` VARCHAR(255) NOT NULL
  - `code` VARCHAR(50) NULL — short code (e.g. PWD, PHED)
  - `description` TEXT NULL
  - `is_active` BOOLEAN NOT NULL DEFAULT TRUE
  - `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
- **Foreign keys:** None
- **Indexes:** `code`, `is_active`

---

## 5. locations

- **Why:** Authority is mapped to geography (e.g. district, pincode). Escalation and routing use `location_id`. Officers and complaints reference it.
- **Primary key:** `location_id` (BIGINT, AUTO_INCREMENT)
- **Columns:**
  - `location_id` BIGINT PRIMARY KEY AUTO_INCREMENT
  - `location_type` VARCHAR(50) NULL — e.g. district, block
  - `name` VARCHAR(255) NOT NULL
  - `code` VARCHAR(50) NULL — e.g. pincode 473551
  - `is_active` BOOLEAN NOT NULL DEFAULT TRUE
  - `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
- **Foreign keys:** None
- **Indexes:** `code`, `is_active`

---

## What is not in this schema

- **No new or changed complaint tables** — complaints, complaint_status_history unchanged.
- **No new or changed escalation tables** — escalation_rules, complaint_escalations unchanged.
- **No new audit tables** — Authority actions recorded in existing `complaint_status_history` (actor_type = 'authority', actor_id = officer_id) and `audit_log` (action_by_type = officer, action_by_officer_id).
- **No role/permission tables** — single Authority type; levels (L1/L2/L3) are derived from `officers.employee_id` pattern, not a separate role table.
- **No department_location_mapping** — optional for pilot; Authority mapping is `officers.department_id` + `officers.location_id`.

---

## Authority levels (L1 / L2 / L3)

- Encoded in **officers.employee_id** (e.g. `PWD-L1-001`, `PHED-L2-001`). Escalation logic derives level from this pattern; no separate level table.
- L1/L2/L3 indicate hierarchy for escalation target lookup only; they are not a permission model.
