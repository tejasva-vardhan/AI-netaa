# Pilot Escalation Rules

Escalation is **authority-based** (department + level), not officer-based. Rules are **admin-owned configuration** (data in `escalation_rules`), not hardcoded. The worker is rule-driven; no logic changes required when adding or changing rules.

---

## 1. Why rules are admin data

- **Behaviour in data**: Change SLA or scope by updating `escalation_rules`, not code.
- **No deployment**: Tune pilot (e.g. 72h → 48h) with a single SQL update.
- **Reproducible**: Same codebase; different environments use different rows.
- **Auditable**: `created_at` / `updated_at` and `is_active` support rollouts and rollbacks.

---

## 2. Meaning of NULL department fields

| Column | NULL meaning |
|--------|----------------|
| `from_department_id` | **Global rule**: applies to complaints from any department. |
| `from_location_id` | **Global rule**: applies to any location. |
| `to_department_id` | **Same-department hierarchy**: target department is not fixed in the rule; it is resolved at runtime from the complaint’s `assigned_department_id`. Escalation stays within the same department (L1→L2→L3). |
| `to_location_id` | **Same location**: keep complaint in same location. |

Pilot uses **global rules**: all four are NULL. No department IDs are hardcoded.

---

## 3. How SLA override works

- **In SQL**: Rules store `sla_hours` in `conditions` JSON (e.g. 72, 120). No minutes in the DB.
- **In code**:  
  - If `TEST_ESCALATION_OVERRIDE_MINUTES > 0`, the worker uses that value (in minutes) as the effective SLA and ignores `sla_hours` for the time check.  
  - Otherwise it uses `sla_hours` (converted to minutes).
- **Result**: Same seed data works for production (hours) and testing (minutes) without changing SQL.

---

## 4. Exact steps to enable fast testing (2-minute escalation)

1. **Apply schema** (once):
   ```bash
   mysql -u USER -p DATABASE < database_migration_fix_escalation_rules.sql
   ```

2. **Seed pilot rules** (once, idempotent):
   ```bash
   mysql -u USER -p DATABASE < seed_escalation_rules_pilot.sql
   ```

3. **Set env and start backend**:
   ```bash
   set TEST_ESCALATION_OVERRIDE_MINUTES=2
   # Start Go server (worker interval auto-adapts to 30s)
   ```

4. **Confirm in logs**:
   - `Test escalation override ENABLED: 2 minutes`
   - `Escalation worker interval: 30 seconds`
   - After ~2 minutes without status change: `Escalated complaint #<id>: ...`

No manual DB edits; no code changes. Reproducible by running the two SQL files and restarting the backend with the env var.

---

## 5. How this scales to multi-city later

- **Same schema**: Add rows to `escalation_rules` with non-NULL `from_department_id` or `from_location_id` (or both) to scope rules to a department/city.
- **Same worker**: It already filters by `ruleMatchesComplaint(rule, candidate)` (department/location). No code change needed to respect new rules.
- **Optional**: `to_department_id` can be set to a specific higher-level department (e.g. regional) when escalation should cross departments. For pilot, NULL = same department hierarchy.

---

## 6. Verification steps (mandatory)

### SQL to run (in order)

```bash
# 1. Migration (makes to_department_id nullable)
mysql -u USER -p DATABASE < database_migration_fix_escalation_rules.sql

# 2. Seed pilot rules (idempotent)
mysql -u USER -p DATABASE < seed_escalation_rules_pilot.sql
```

### Query to verify rules exist

```sql
SELECT rule_id, escalation_level, from_department_id, to_department_id, is_active,
       JSON_EXTRACT(conditions, '$.time_based.sla_hours') AS sla_hours
FROM escalation_rules
WHERE is_active = 1
ORDER BY escalation_level;
```

Expected: two rows — `escalation_level` 0 and 1; both `from_department_id` and `to_department_id` NULL; `sla_hours` 72 and 120.

### Log line that confirms escalation fired

- `Escalated complaint #<complaint_id>: <reason>`  
  (from `worker/escalation_worker.go` after a successful escalation.)

### DB change that confirms success

- New row in `complaint_escalations` for that complaint (and optionally in `complaint_status_history` with new status `escalated`).

---

## 7. Confirmation checklist

- [ ] Migration run: `to_department_id` is NULLABLE (no error on seed).
- [ ] Seed run: Two active global rules (level 0 and 1), NULL departments.
- [ ] Backend reads rules: No "No rules configured" when rules exist.
- [ ] With `TEST_ESCALATION_OVERRIDE_MINUTES=2`: Log shows "Test escalation override ENABLED: 2 minutes" and worker interval 30s.
- [ ] After ~2 minutes: Log shows "Escalated complaint #..."; `complaint_escalations` has a new row.
- [ ] No manual DB edits required after migration + seed.
- [ ] Escalation is authority-based (department + level); no officer IDs in rules.
