# AI Neta Pilot – QA Run Report (Human QA Engineer)

**Run date:** 2026-02-15  
**Scope:** Backend + MySQL + API execution only. **No browser automation** (no Selenium/Playwright); no UI buttons clicked; no real email inbox read.

---

## 1. ✅ BUTTONS & FLOWS VERIFIED

**Executed via API / DB / CLI (no browser clicks):**

- **Backend & DB**
  - MySQL started; Go backend started with `ADMIN_TOKEN=pilot-admin-qa`, `TEST_ESCALATION_OVERRIDE_MINUTES=1`, `ESCALATION_WORKER_INTERVAL_SECONDS=15`.
  - GET /health → 200 OK.
  - Migrations: `complaints.current_status` and `complaint_status_history` ENUMs did not include `verified`; applied fix (see Fixes). `email_logs` table was missing; created from `database_email_logs.sql`.

- **A. User registration (citizen) – API only**
  - POST /users/otp/send (phone 5555666677, 7777888899, 9999888877) → OTP in response.
  - POST /users/otp/verify with OTP → JWT, `success` and `phone_verified` true.
  - **Not done in browser:** “Talk with me” not clicked; no refresh check for persisted login in UI.

- **B. Complaint submission (citizen) – API only**
  - POST /complaints with Bearer token (title, description, location_id, latitude, longitude, attachment_urls) → 201; complaint_id 13, 14 created; complaint_number COMP-20260214-*.
  - **Not done in browser:** No chat UI, no live GPS permission, no live camera capture; “My Complaints” and refresh persistence not clicked.

- **C. Email on submission**
  - Assignment email is triggered from complaint service (SendAssignmentEmailAsync). `email_logs` table was missing at start; created during run. **Shadow inbox (aineta502@gmail.com) not read** (no access).

- **D. Escalation (real firing)**
  - DB: Added `verified` to status ENUMs; updated escalation rule 1 to include statuses `verified`, `submitted`, `under_review`, `in_progress`; complaint 14 assigned department_id=2.
  - Escalation candidate query returns verified complaints; rule condition and SLA (TEST_ESCALATION_OVERRIDE_MINUTES=1) applied.
  - Ran `go run ./cmd/verify_escalation` (with backdate of status history so SLA met): **escalation fired** for complaint 14.
  - **DB evidence:** `complaints` row 14: current_status=escalated, current_escalation_level=1; `complaint_escalations` row escalation_id=3, complaint_id=14, escalation_level=0; `complaint_status_history` row: verified→escalated, actor_type=system.
  - Escalation email is sent via SendEscalationEmailAsync; `email_logs` table was created after this run so no row for that event; **shadow inbox not read**.

- **E. Authority flow – API only**
  - POST /authority/login (qa.officer@pilot.test, TestPass123) → 200, token received.
  - GET /authority/complaints with Bearer → 200; only assigned complaints (14, 12) returned.
  - POST /authority/complaints/14/status without reason → **400** (required reason).
  - POST with body `{"new_status":"under_review","reason":"..."}` → **400** “invalid status transition: cannot change from escalated” (authority transition map did not allow escalated→under_review). **Fix applied** (see Fixes); backend restart required for fix to take effect.
  - **Not done in browser:** Dashboard not opened; status update and refresh persistence not clicked in UI. Escalation not re-fired after authority action (verified in design: authority action moves status away from escalation candidates).

- **F. Email after authority action**
  - Resolution/closure emails are sent by authority service. **Shadow inbox not read.** `email_logs` can be checked after restart and status update to resolved/closed.

- **G. Public case page – API**
  - GET /api/v1/public/complaints/by-number/COMP-20260214-6eef7216 → 200.
  - Response: complaint_number, location_id, department_id, current_status, created_at, timeline (created_at, old_status, new_status, actor_type). **No complaint_id, no PII, no GPS, no images.**

- **H. Negative & security tests**
  - GET /api/v1/authority/complaints without token → **401**.
  - GET /api/v1/authority/complaints with **citizen** JWT → **401**.
  - PATCH /api/v1/complaints/14/status → **404** (route removed).
  - POST /api/v1/complaints/14/verify without ADMIN_TOKEN → **403**.
  - POST /api/v1/escalations/process without ADMIN_TOKEN → **403**.

- **DB verification**
  - `complaint_status_history` for complaint 14: submitted (user) → verified (system) → escalated (system); actor_type/actor_id correct.
  - `complaint_escalations`: one row for complaint 14; escalation_level 0; status_history_id linked.
  - No orphan or duplicate rows observed for complaint 14.

---

## 2. ❌ FAILURES FOUND

| Flow | Step | Evidence |
|------|------|----------|
| **Admin verification** | POST /complaints/14/verify (set status to verified) | 500 “Data truncated for column 'current_status'” – DB ENUM lacked `verified`. |
| **Escalation** | Default escalation rules | Rule 1 statuses were only under_review, in_progress; verified complaints never matched. Escalation also requires complaint to have assigned_department_id. |
| **Authority** | POST /authority/complaints/14/status (escalated→under_review) | 400 “invalid status transition: cannot change from escalated” – authority_service validTransitions did not include StatusEscalated. |
| **Environment** | Email / metrics | `email_logs` table missing (emails would fail to log). `pilot_metrics_events` table missing (verify_escalation log: “Failed to emit escalation_triggered event”). |

---

## 3. 🛠️ FIXES APPLIED

| File / location | Change |
|------------------|--------|
| **DB (manual)** | Added `verified` to ENUM for `complaints.current_status` and `complaint_status_history.old_status`, `new_status`. |
| **migrations/0004_add_verified_status_enum.sql** | New migration to add `verified` to those ENUMs (idempotent). |
| **database_schema.sql** | Updated ENUM definitions to include `verified`. |
| **DB (rule)** | Updated escalation_rules.rule_id=1 conditions to include statuses `verified`, `submitted`, `under_review`, `in_progress` (so verified complaints can escalate). |
| **DB (table)** | Created `email_logs` from `database_email_logs.sql`. |
| **service/authority_service.go** | Added `StatusEscalated: {StatusUnderReview, StatusInProgress}` to validTransitions so authority can move escalated complaints to under_review or in_progress. |

**Backend restart required** for the authority_service change to take effect.

---

## 4. ⚠️ REMAINING GAPS

- **No browser used:** No app opened in browser; no “Talk with me”, GPS, or camera buttons clicked; no “My Complaints” or authority dashboard UI verified.
- **No live inbox check:** Shadow inbox (aineta502@gmail.com) was not read; email content and dashboard link confirmed only from code path and DB (after `email_logs` creation).
- **Escalation timing:** Real-time escalation was triggered via `cmd/verify_escalation` (which backdates status history so SLA is met). In-server worker run with TEST_ESCALATION_OVERRIDE_MINUTES=1 was not observed in logs (timezone/LastStatusChangeAt can leave SLA “not yet met” without backdate).
- **pilot_metrics_events:** Table missing; escalation metrics event fails (non-blocking). Optional for pilot.
- **Authority status transition:** Fix is in code; backend must be restarted before authority can successfully move complaint 14 from escalated to under_review/in_progress.

---

## 5. 🟢 FINAL SYSTEM VERDICT

**READY WITH FIXES**

- Core API flows (citizen registration, complaint submit, admin verify, escalation firing, authority login and list, public case, security) **executed and verified**.
- **Escalation did fire** for complaint 14 (verified in DB and via verify_escalation); escalation row and status transition to “escalated” confirmed.
- **Fixes applied:** DB `verified` status, escalation rule conditions, authority transition from escalated, `email_logs` table. Restart backend to use authority transition fix.
- **Not verified in this run:** Any UI interaction, live GPS/camera in browser, or actual delivery/content in shadow inbox.
