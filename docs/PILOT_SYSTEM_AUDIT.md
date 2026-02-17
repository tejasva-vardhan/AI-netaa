# AI Neta Pilot – Full System Audit

## A. ✅ CONFIRMED WORKING

- **Citizen auth:** JWT in `Authorization: Bearer`; `user_id` set in context; GetUserComplaints, GetComplaintByID, GetStatusTimeline, CreateComplaint all require auth and use context `user_id`.
- **Ownership enforcement:** GetComplaintByID and GetStatusTimeline check `complaint.UserID != requestingUserID && !complaint.IsPublic`; 404/access denied for non-owners of non-public complaints.
- **Authority auth:** Separate JWT with `actor_type: "authority"` and `officer_id`; `authority_token` in frontend; authority routes use `RequireAuthorityAuth`; GetMyComplaints and UpdateComplaintStatus/AddNote use officer from context.
- **Authority assignment check:** UpdateComplaintStatus (authority) verifies complaint is assigned to the logged-in officer before allowing status change.
- **Authority status transitions:** Allowed path enforced in service (e.g. under_review → in_progress → resolved); mandatory reason; closed is system-only.
- **Admin auth:** Env-based `ADMIN_TOKEN`; 403 on missing/mismatch; admin routes isolated; no citizen/authority token used for admin.
- **Public case page:** Fetches by `complaint_number` only; response has no `complaint_id`; whitelist: complaint_number, location_id, department_id, current_status, created_at, timeline (created_at, old_status, new_status, actor_type); no PII, GPS, notes, actor_id.
- **Escalation flow:** Candidates filtered by status (verified, under_review, in_progress); escalated complaints excluded from next run; SLA uses `LastStatusChangeAt` from `complaint_status_history`; `now` in UTC; idempotency via `HasExistingEscalation(complaintID, level, 1 hour)`.
- **Escalation worker:** ProcessEscalations is safe to call repeatedly; no double-escalation at same level within window.
- **Email shadow mode:** All authority emails to PilotInboxEmail; deep link uses path/FRONTEND_URL; “Login required”; no token in URL; no action-by-email.
- **Authority passwords:** Stored via bcrypt (HashAuthorityPassword); login uses CheckAuthorityPassword; no plaintext.
- **Audit:** Complaint create, authority status update, admin create/update officer write to `audit_log` with action_by_type; escalation/reminder logged with ActorSystem.
- **DB connectivity:** DSN uses `parseTime=true&loc=UTC`; schema has indexes on complaint_id, user_id, current_status, complaint_id+created_at for status history.
- **Frontend citizen flow:** api.js uses `auth_token`; ComplaintDetailScreen handles loading/error/retry; ComplaintsListScreen uses same API.
- **Frontend authority flow:** authorityApi.js uses `authority_token`; status update sends new_status + reason; dashboard lists complaints from API.

---

## B. ⚠️ ISSUES FOUND

### 1. PATCH /api/v1/complaints/{id}/status – no authentication
- **Severity:** HIGH  
- **Area:** Backend / Security  
- **Location:** `routes/routes.go` line 69; `handler/complaint_handler.go` `UpdateComplaintStatus`  
- **What can break:** Any client can change any complaint’s status, assignment, and timestamps by sending PATCH with forged `X-Actor-Type`, `X-User-ID`, `X-Officer-ID`.  
- **Reproduction:** `curl -X PATCH http://localhost:8080/api/v1/complaints/1/status -H "Content-Type: application/json" -d '{"new_status":"resolved"}'` (no auth; actor from headers).

### 2. POST /api/v1/complaints/{id}/verify – no authentication
- **Severity:** MEDIUM  
- **Area:** Backend / Security  
- **Location:** `routes/routes.go` line 72; `handler/verification_handler.go` `VerifyComplaint`  
- **What can break:** Anyone who knows a complaint_id can trigger verification (e.g. draft → submitted), affecting lifecycle and escalation eligibility.  
- **Reproduction:** `curl -X POST http://localhost:8080/api/v1/complaints/1/verify` (no auth).

### 3. POST /api/v1/escalations/process – no authentication
- **Severity:** MEDIUM  
- **Area:** Backend / Security  
- **Location:** `routes/routes.go` line 89; `handler/escalation_handler.go` `ProcessEscalations`  
- **What can break:** Anyone can trigger a full escalation run (stress, repeated runs, or abuse in production).  
- **Reproduction:** `curl -X POST http://localhost:8080/api/v1/escalations/process`.

### 4. Escalation: status update vs history/escalation record not atomic
- **Severity:** MEDIUM  
- **Area:** Backend / DB  
- **Location:** `service/escalation_service.go` `executeEscalation`: `UpdateComplaintStatus` then `CreateStatusHistory` then `CreateEscalation`.  
- **What can break:** If CreateStatusHistory or CreateEscalation fails after UpdateComplaintStatus, complaint can be left with status `escalated` but no matching history row or escalation row (audit/timeline inconsistency).  
- **Reproduction:** Simulate DB error or kill process after first Exec and before CreateStatusHistory.

### 5. Authority complaint detail: refresh / direct URL loses state
- **Severity:** MEDIUM  
- **Area:** Frontend  
- **Location:** `frontend/src/screens/authority/AuthorityComplaintDetailScreen.jsx` – relies on `location.state?.complaint`.  
- **What can break:** Navigating directly to `/authority/complaints/123` or refreshing shows “Complaint not found. Go back to list.” with no refetch by id.  
- **Reproduction:** Open a complaint from dashboard, then refresh or paste `/authority/complaints/<id>` in URL.

### 6. JWT_SECRET default weak when unset
- **Severity:** LOW  
- **Area:** Security  
- **Location:** `routes/routes.go` lines 38–40: `jwtSecret = "pilot-secret-key-change-in-production"` when `JWT_SECRET` is empty.  
- **What can break:** If env is not set in production, tokens are signed with a known default and could be forged.  
- **Reproduction:** Deploy without JWT_SECRET; tokens remain valid and predictable.

### 7. Assignment email body includes complaint_id
- **Severity:** LOW  
- **Area:** Email  
- **Location:** `service/email_shadow_service.go` – body contains “Complaint ID: %s” with complaint_id.  
- **What can break:** complaint_id is exposed in text sent to pilot inbox; acceptable for internal pilot but diverges from “complaint_id must not be exposed publicly” if email is ever forwarded or leaked.

### 8. Citizen timeline response includes changed_by_user_id / changed_by_officer_id
- **Severity:** LOW  
- **Area:** Backend / Security  
- **Location:** `service/complaint_service.go` GetStatusTimeline – timeline entries include `ChangedByUserID`, `ChangedByOfficerID` when valid.  
- **What can break:** Numeric user/officer IDs are visible to the complaint owner; low PII risk but identifiable if combined with other data.

---

## C. 🛠️ SUGGESTED FIXES (OPTIONAL)

1. **PATCH /complaints/{id}/status:** Remove from public API or protect with a dedicated internal/system token or network restriction. If kept, require a valid JWT (e.g. authority or admin) or a separate service-to-service secret and stop trusting `X-Actor-Type` / `X-User-ID` / `X-Officer-ID` from untrusted clients.

2. **POST /complaints/{id}/verify:** Require auth (e.g. citizen owner or internal token) or restrict to internal/service calls only.

3. **POST /escalations/process:** Require admin token (e.g. same as `ADMIN_TOKEN`) or restrict to internal network / specific IP so only operators can trigger manual runs.

4. **Escalation atomicity:** Wrap `UpdateComplaintStatus`, `CreateStatusHistory`, and `CreateEscalation` (and any related escalation-level update) in a single DB transaction in `executeEscalation`; commit only after all steps succeed.

5. **Authority detail refresh:** In AuthorityComplaintDetailScreen, when `location.state?.complaint` is missing but `id` from URL is present, fetch complaint by id using authority API (e.g. GET complaint by id for assigned officer) and render from that; show loading/error states.

6. **JWT_SECRET:** In production, require JWT_SECRET to be set (log warning or fail startup if empty) so the default is never used.

7. **Email body:** For pilot, optionally replace “Complaint ID: …” with “Complaint number: …” only so complaint_id does not appear in text (dashboard link can still use id server-side if needed).

8. **Timeline PII:** If desired, omit `changed_by_user_id` and `changed_by_officer_id` from citizen timeline response and keep only `actor_type` (e.g. user/authority/system).

---

## D. 🟢 PILOT READINESS VERDICT

**READY WITH FIXES**

The system is structurally sound for the pilot: citizen and authority flows are separated and correctly enforced, escalation logic and idempotency are in place, public page uses only complaint_number and a strict whitelist, and email is notification-only in shadow mode. The main blocker for production-like deployment is **unauthenticated PATCH /complaints/{id}/status**, which allows arbitrary status and assignment changes. Addressing that (and optionally securing verify and escalation/process, plus the escalation transaction and authority-detail refresh) would make the pilot ready for controlled rollout. Applying the suggested fixes in section C in order of severity is recommended before opening to real users or less trusted networks.
