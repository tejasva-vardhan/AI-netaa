-- ============================================================================
-- PILOT ESCALATION RULES SEED (admin-owned configuration)
-- ============================================================================
--
-- Prerequisite: Run database_migration_fix_escalation_rules.sql first so
-- to_department_id is NULLABLE. Otherwise INSERT with NULL will fail.
--
-- These rules are ADMIN DATA: change behaviour by editing data, not code.
-- Escalation is authority-based (department + level), not officer-based.
--
-- Escalation level = CURRENT level before escalation:
--   escalation_level 0 → when at L1, escalate to L2 (sla_hours = 72)
--   escalation_level 1 → when at L2, escalate to L3 (sla_hours = 120)
--   escalation_level 2 = L3 max (no rule; worker enforces in code)
--
-- SLA: Stored as sla_hours in conditions JSON. TEST_ESCALATION_OVERRIDE_MINUTES
-- (env) overrides sla_hours in code for fast testing (e.g. 2 = ~2 min).
--
-- Scope: GLOBAL (from_department_id = NULL, to_department_id = NULL).
-- No hardcoded department IDs. NULL to_department_id = same department hierarchy.
--
-- ============================================================================

-- Rule 1: L1 → L2 escalation (72 hours SLA)
-- Idempotent: Only insert if rule doesn't already exist
INSERT INTO escalation_rules (
    from_department_id,
    from_location_id,
    to_department_id,
    to_location_id,
    escalation_level,
    conditions,
    is_active,
    created_at
)
SELECT
    NULL, -- Any source department (global rule)
    NULL, -- Any source location
    NULL, -- Target: NULL = escalate within same department hierarchy
    NULL, -- Same location
    0, -- Escalation level: when at L1 (level 0), escalate to L2
    JSON_OBJECT(
        'time_based', JSON_OBJECT(
            'sla_hours', 72
        ),
        'statuses', JSON_ARRAY('under_review', 'in_progress')
    ),
    TRUE,
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM escalation_rules
    WHERE escalation_level = 0
      AND from_department_id IS NULL
      AND from_location_id IS NULL
      AND to_department_id IS NULL
      AND to_location_id IS NULL
      AND is_active = TRUE
);

-- Rule 2: L2 → L3 escalation (120 hours SLA)
-- Idempotent: Only insert if rule doesn't already exist
INSERT INTO escalation_rules (
    from_department_id,
    from_location_id,
    to_department_id,
    to_location_id,
    escalation_level,
    conditions,
    is_active,
    created_at
)
SELECT
    NULL, -- Any source department (global rule)
    NULL, -- Any source location
    NULL, -- Target: NULL = escalate within same department hierarchy
    NULL, -- Same location
    1, -- Escalation level: when at L2 (level 1), escalate to L3
    JSON_OBJECT(
        'time_based', JSON_OBJECT(
            'sla_hours', 120
        ),
        'statuses', JSON_ARRAY('under_review', 'in_progress')
    ),
    TRUE,
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM escalation_rules
    WHERE escalation_level = 1
      AND from_department_id IS NULL
      AND from_location_id IS NULL
      AND to_department_id IS NULL
      AND to_location_id IS NULL
      AND is_active = TRUE
);

-- Note: No rule for L3 → L4 (escalation_level = 2)
-- The escalation worker enforces max level programmatically (currentLevel >= 2 = skip)

-- ============================================================================
-- HOW WORKER MATCHES RULES:
-- ============================================================================
-- 1. Worker loads all active rules: SELECT * FROM escalation_rules WHERE is_active = TRUE
-- 2. For each complaint candidate:
--    a. Gets current escalation level (0=L1, 1=L2, 2=L3)
--    b. Finds rules where escalation_level == currentLevel
--    c. Checks if rule matches complaint (department/location filters)
--    d. Evaluates conditions (sla_hours from JSON)
--    e. If TEST_ESCALATION_OVERRIDE_MINUTES > 0, code overrides sla_hours with minutes
-- 3. If conditions met, executes escalation
--
-- TEST OVERRIDE EXAMPLE:
-- - TEST_ESCALATION_OVERRIDE_MINUTES=2
-- - Rule has sla_hours=72 in JSON
-- - Code overrides: effective_sla = 2 minutes (not 72 hours)
-- - Escalation triggers after 2 minutes (no SQL change needed)
-- ============================================================================
