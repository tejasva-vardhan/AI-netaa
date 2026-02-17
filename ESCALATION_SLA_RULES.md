# Escalation SLA Rules (Pilot)

Deterministic, rule-based escalation system with SLA time limits. Maximum escalation level is **L3** (no escalation beyond L3).

**Escalation reassigns authority (department), not personnel (officers).**

## SLA Escalation Rules

| Escalation | SLA Hours | Escalation Level | Conditions |
|------------|-----------|------------------|------------|
| **L1 → L2** | 72 hours | Level 0 | When complaint is at L1 (current level 0), escalate to L2 |
| **L2 → L3** | 120 hours | Level 1 | When complaint is at L2 (current level 1), escalate to L3 |
| **L3** | Max level | N/A | No escalation beyond L3 |

**Escalation level semantics:**
- `escalation_level` refers to **CURRENT level before escalation**
- Level 0 = L1 (initial assignment, no escalation yet)
  - Rule with `escalation_level = 0` means: "when complaint is at L1, escalate to L2"
- Level 1 = L2 (first escalation completed)
  - Rule with `escalation_level = 1` means: "when complaint is at L2, escalate to L3"
- Level 2 = L3 (second escalation completed, maximum)

## Database Seed

**File:** `seed_escalation_rules_sla.sql`

Contains two escalation rules:
1. **L1 → L2**: `escalation_level = 0` (current level L1), `sla_hours = 72`
2. **L2 → L3**: `escalation_level = 1` (current level L2), `sla_hours = 120`

**Department rules:**
- `from_department_id = NULL`: Global rules (apply to any department)
- `to_department_id = NULL`: Escalate within same department hierarchy (authority lookup resolves target)
- For department-specific escalation targets, set `to_department_id` to target department ID

**Apply:** Run `seed_escalation_rules_sla.sql` on your MySQL database to insert pilot SLA rules.

## Worker Safeguards

The escalation worker (`service/escalation_service.go`) enforces:

### 1. Maximum Level Enforcement
- **Location:** `processComplaintEscalation()`
- **Logic:** If `currentLevel >= 2` (L3), escalation is skipped
- **Logging:** Logs `"[ESCALATION] Complaint {id} already at max escalation level L3 (no further escalation possible)"`
- **Behavior:** Returns `nil, nil` to skip escalation silently (worker continues processing other complaints)

### 2. Authority Existence Check
- **Location:** `executeEscalation()`
- **Logic:** Before escalating, checks if authority exists at target department + location + escalation level
- **Authority lookup:** Uses `FindAuthorityByDepartmentPincodeLevel()` (department_id + location_id + escalation_level)
- **If no authority found:** Logs warning (not error) and skips escalation
- **Warning message:** `"[ESCALATION] Warning: no authority exists at escalation level {level} (target L{target}) for department {dept}, location {loc} - skipping escalation for complaint {id}"`
- **Behavior:** Returns `nil, nil` (not an error) - worker remains healthy and continues processing other complaints

## Escalation Worker Behavior

- **Respects SLA hours:** Uses `sla_hours` from escalation rule conditions (backward compatible with `hours_since_status_change`)
- **Does NOT escalate beyond L3:** Enforced by safeguard check
- **Logs max level reached:** When complaint is already at L3, logs and skips
- **Skips if no authority:** When no authority exists at next level, logs warning (not error) and skips
- **Non-blocking:** Warnings/errors in one complaint do not stop processing of other complaints
- **Worker health:** Escalation worker remains healthy even when authorities are missing (logs warnings, continues processing)

## Escalation Flow

1. Worker loads active escalation rules from `escalation_rules` table
2. Gets escalation candidates (complaints in `under_review`, `in_progress` statuses)
3. For each candidate:
   - Gets current escalation level (0 = L1, 1 = L2, 2 = L3)
   - **Safeguard:** If at L3, log and skip
   - Finds applicable rules where `escalation_level` matches current level
   - Evaluates SLA conditions (`sla_hours` - time since last status change)
   - **Safeguard:** Checks if authority exists at target department + location + level
     - Authority lookup: `FindAuthorityByDepartmentPincodeLevel(department_id, location_id, escalation_level)`
     - If no authority found: log warning, skip escalation (worker continues)
   - If conditions met and authority exists, executes escalation
4. Creates escalation record, updates complaint assignment (authority reassignment), sends email (shadow mode)

## Notes

- **Deterministic:** Same inputs always produce same escalation path
- **Rule-driven:** No AI or heuristics; purely based on configured rules and SLA hours
- **Authority-based:** Escalation reassigns authority (department), not personnel (officers)
- **No placeholder IDs:** Rules use `NULL` for global rules or real department IDs; no placeholder department IDs
- **Escalation level semantics:** `escalation_level` in rules refers to CURRENT level before escalation, not target level
