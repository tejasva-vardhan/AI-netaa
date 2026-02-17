# Email Shadow Mode (Pilot)

All authority emails (assignment, escalation, resolution) are sent **only** to the pilot inbox **aineta502@gmail.com**. Real authority addresses are not used. Full email content is still generated (intended authority level, department, complaint ID) and stored in `email_logs`.

**Authority abstraction:** Emails represent authority-level notifications (department + level L1/L2/L3), not officer-based. Officers are operational actors, not escalation or notification targets.

## 1. DB schema: `email_logs`

**File:** `database_email_logs.sql`

```sql
CREATE TABLE IF NOT EXISTS email_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    complaint_id BIGINT NOT NULL,
    email_type ENUM('assignment','escalation','resolution') NOT NULL,
    intended_authority_id BIGINT NULL COMMENT 'Authority identifier (for authority abstraction, not officer)',
    intended_level VARCHAR(10) NULL COMMENT 'L1/L2/L3 escalation level',
    department_id BIGINT NOT NULL COMMENT 'Department ID (explicit authority department)',
    sent_to_email VARCHAR(255) NOT NULL,
    subject VARCHAR(500) NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_complaint_id (complaint_id),
    INDEX idx_email_type (email_type),
    INDEX idx_department_id (department_id),
    INDEX idx_created_at (created_at)
);
```

**Apply:** Run `database_email_logs.sql` on your MySQL database before using shadow mode.

## 2. Email service changes

- **`service/email_shadow_service.go`** (new)
  - `PilotInboxEmail = "aineta502@gmail.com"`.
  - `SendAssignmentEmailAsync(complaintID, complaintNumber, departmentID, departmentName)` – builds subject/body with authority level (L1) and department, logs to `email_logs`, sends to pilot inbox in a goroutine. **Authority abstraction:** Uses `department_id` only, not officer.
  - `SendEscalationEmailAsync(complaintID, complaintNumber, escalationLevel, toDepartmentID, departmentName, reason)` – same pattern for escalation (L1/L2/L3). **Authority abstraction:** Uses `to_department_id` + level, not officer.
  - `SendResolutionEmailAsync(complaintID, complaintNumber, departmentID, departmentName, newStatus, reason)` – for resolution/closure. **Authority abstraction:** Uses `department_id` only, not officer.
  - All send paths: **log first** (insert `email_logs` with `department_id`), then **send via** `notification.EmailSender` to `PilotInboxEmail` in a **goroutine**. Failures in log or send are logged and **do not** affect the complaint flow.

- **`repository/email_log_repository.go`** (new) – `Create(log *models.EmailLog)` inserts into `email_logs` (includes `department_id`).

- **`models/entities.go`** – added `EmailLogType` (`assignment` / `escalation` / `resolution`) and `EmailLog` struct for `email_logs` (includes `DepartmentID`).

## 3. Where email send is triggered

| Event            | Location                               | Method / trigger |
|-----------------|----------------------------------------|-------------------|
| **Assignment**   | `service/complaint_service.go`         | After `CreateComplaint` and status history; **always** when `AssignedDepartmentID` is set (no check on `AssignedOfficerID`). `emailShadowService.SendAssignmentEmailAsync(complaintID, complaintNumber, departmentID, departmentName)` is called. |
| **Escalation**   | `service/escalation_service.go`        | In `executeEscalation`, after `CreateEscalation` and audit log; `emailShadowService.SendEscalationEmailAsync(...)` with candidate complaint number, rule level, **to_department_id** (not officer), department name, reason. |
| **Resolution**   | `service/authority_service.go`         | In `UpdateComplaintStatus`, after status and audit log; when `newStatus` is `resolved` or `closed`, `emailShadowService.SendResolutionEmailAsync(...)` with complaint number, **department_id** (not officer), department name, new status, reason. |

All three triggers use the shared **email shadow service**; it is injected into `ComplaintService`, `EscalationService`, and `AuthorityService` from `main.go` and `routes.SetupRoutes`.

**Email trigger conditions:**
- **Assignment email:** Always on assignment to an authority (when `complaints.assigned_department_id` is set).
- **Escalation email:** Always on authority reassignment (when escalation moves complaint to a new department).
- **Resolution email:** On resolved / closed by authority (when status becomes resolved or closed).
- **No conditional checks on officer fields:** `AssignedOfficerID` is not referenced in email triggering.

## 4. Behaviour summary

- **Authority abstraction:** Emails represent authority-level notifications (department + level L1/L2/L3), not officer-based. Officers are operational actors, not escalation or notification targets.
- **Content:** Subject and body include complaint ID/number, intended authority level (L1/L2/L3), department (e.g. “Department &lt;id&gt;”), and reason where applicable. **No officer names or IDs** are used in email content.
- **Recipient:** Every such email is sent only to **aineta502@gmail.com** (no real authority addresses).
- **Persistence:** Each email is stored in `email_logs` (complaint_id, email_type, intended_authority_id, intended_level, **department_id**, sent_to_email, subject, body, created_at). `intended_authority_id` represents the authority abstraction (department), not an officer.
- **Resilience:** Email logging and sending run in goroutines where applicable; errors are logged and do not block or fail complaint creation, escalation, or status update.

## 5. No frontend changes

No frontend or API contract changes; behaviour is backend-only.
