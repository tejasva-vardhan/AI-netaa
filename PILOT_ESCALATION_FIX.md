# Pilot Escalation Fix – Summary & Verification

## Root cause (1 paragraph)

Escalation did not fire because **the candidate query in `GetEscalationCandidates` only returned complaints that were “stale”** (no update or no status change in the last 24 hours). Complaint 10 had a recent status change (SLA exceeded by >30 minutes but within 24h), so it was never selected as a candidate. In addition, authority lookup could fail when no officer matched the `employee_id` pattern (e.g. L2) for the complaint’s department+location; with no fallback, escalation was skipped. Fixes: (1) **Candidate query** – removed the 24h time filter; candidates are now all non-terminal status complaints; SLA is applied later in `evaluateEscalationConditions` (including `TEST_ESCALATION_OVERRIDE_MINUTES=2`). (2) **Authority lookup** – try pattern first, then **any active officer** in the same department+location (pilot fallback). (3) **Pilot-only safeguard** – if SLA and rule are satisfied but authority lookup returns no officer, **escalate anyway** (increment level, keep department, officer unassigned). (4) **DEBUG logging** added for candidates, rules, skip reasons, authority inputs/result, and a single “ESCALATION FIRED” log line for proof.

---

## Files changed

| File | Changes |
|------|--------|
| `repository/escalation_repository.go` | `GetEscalationCandidates`: removed 24h cutoff; only status filter. `FindAuthorityByDepartmentPincodeLevel`: try employee_id pattern first, then any active officer in dept+location; added `[ESCALATION_DEBUG]` logs (inputs, SQL shape, rows returned). |
| `service/escalation_service.go` | `ProcessEscalations`: log rule count and per-rule level; candidate count and per-candidate (complaint_id, status, department_id, pincode, location_id, minutes_since_status_change). `processComplaintEscalation`: log current_escalation_level; skip reasons (max level, no rule, SLA not satisfied). `executeEscalation`: log authority inputs; when authority is nil, log WARNING and **escalate anyway** (PILOT); after `CreateEscalation`, call `UpdateComplaintEscalationLevel(complaintID, newLevel)` then log **`[ESCALATION] ESCALATION FIRED complaint_id=%d new_escalation_level=%d (from %d) escalation_id=%d`**. |
| `repository/complaint_repository.go` | **`UpdateComplaintEscalationLevel(complaintID, level)`** – sets `complaints.current_escalation_level` so it becomes 1 after first escalation (column must exist). |

---

## Key logs added

- **`[ESCALATION_DEBUG] Loaded N active rules`** and **`Rule i: escalation_level=N`**
- **`[ESCALATION_DEBUG] Fetched N escalation candidates`**
- **`[ESCALATION_DEBUG] Candidate complaint_id=... current_status=... department_id=... pincode=... location_id=... minutes_since_status_change=...`**
- **`[ESCALATION_DEBUG] complaint_id=... current_escalation_level=...`**
- **`[ESCALATION_DEBUG] skip complaint N: no rule found ...`** | **max escalation level** | **SLA/conditions not satisfied - ...**
- **`[ESCALATION_DEBUG] FindAuthorityByDepartmentPincodeLevel inputs: department_id=... location_id=... current_level=...`**
- **`[ESCALATION_DEBUG] Authority lookup (pattern): rows_returned=0/1`** / **Authority lookup (any): rows_returned=0/1**
- **`[ESCALATION] WARNING [PILOT] No authority found for complaint N - escalating anyway ...`** (when escalating without officer)
- **`[ESCALATION] ESCALATION FIRED complaint_id=N new_escalation_level=1 (from 0) escalation_id=M`** ← **proof line** (newLevel and from level in log)

---

## Restart backend

- **Windows (PowerShell):** stop the running backend (Ctrl+C or stop the process), then start again, e.g. `go run .` or your usual run command from the project root.
- **Linux/macOS:** same – restart the Go process that runs the API and escalation worker (worker runs every 30s inside the same process).

No DB migrations were required; only code and logging changes.

---

## Final verification (mandatory)

After deploying and restarting:

1. **Log proof**  
   Within at most two worker cycles (e.g. 60s), you should see:
   ```text
   [ESCALATION] ESCALATION FIRED complaint_id=10 new_escalation_level=1 (from 0) escalation_id=<id>
   ```

2. **DB proof**  
   Run:
   ```sql
   SELECT complaint_id, current_status, assigned_department_id, current_escalation_level
   FROM complaints WHERE complaint_id = 10;

   SELECT escalation_id, complaint_id, escalation_level, to_department_id, created_at
   FROM complaint_escalations WHERE complaint_id = 10 ORDER BY created_at DESC LIMIT 1;
   ```
   Expected: **`complaints.current_escalation_level` = 1** (set by `UpdateComplaintEscalationLevel`; if the column is missing, you’ll see a warning in logs but escalation still fires). **`complaint_escalations`** has one row for complaint_id=10 with `escalation_level=0` (from L1). **`complaints.current_status`** = `escalated`. The **exact log line** above is the definitive proof that escalation fired for complaint 10.

---

## Pilot-only safeguards (remove post-pilot)

- In `executeEscalation`: the branch that logs **`[ESCALATION] WARNING [PILOT] No authority found ... escalating anyway`** and sets `toOfficerID = nil` to still perform escalation.
- Optionally strip or gate all **`[ESCALATION_DEBUG]`** logs once the pilot is stable.
