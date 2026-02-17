-- ============================================================================
-- PILOT SLA ESCALATION RULES
-- ============================================================================
-- 
-- Escalation hierarchy: L1 → L2 → L3 (max level)
-- SLA-based, deterministic, rule-driven escalation
--
-- Rules:
-- - L1 → L2: 72 hours SLA (escalation_level = 0 means "when at L1, escalate")
-- - L2 → L3: 120 hours SLA (escalation_level = 1 means "when at L2, escalate")
-- - No escalation beyond L3
--
-- Escalation level semantics:
-- - escalation_level refers to CURRENT level before escalation
-- - escalation_level = 0: when complaint is at L1 (level 0), escalate to L2
-- - escalation_level = 1: when complaint is at L2 (level 1), escalate to L3
--
-- ============================================================================

-- Clear existing pilot escalation rules (if any)
DELETE FROM escalation_rules WHERE is_active = TRUE;

-- Rule 1: L1 → L2 escalation (72 hours SLA)
-- escalation_level = 0 means: when complaint is at L1 (current level 0), escalate to L2
-- from_department_id = NULL means: applies to any department (global rule)
-- to_department_id = NULL means: escalate within same department hierarchy (or use department-specific rules)
-- Conditions: sla_hours = 72 (escalate if no status change in 72 hours)
INSERT INTO escalation_rules (
    from_department_id,
    from_location_id,
    to_department_id,
    to_location_id,
    escalation_level,
    conditions,
    is_active,
    created_at
) VALUES (
    NULL, -- Any source department (global rule)
    NULL, -- Any source location
    NULL, -- Target: NULL = escalate within same department hierarchy (authority lookup will resolve)
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
);

-- Rule 2: L2 → L3 escalation (120 hours SLA)
-- escalation_level = 1 means: when complaint is at L2 (current level 1), escalate to L3
INSERT INTO escalation_rules (
    from_department_id,
    from_location_id,
    to_department_id,
    to_location_id,
    escalation_level,
    conditions,
    is_active,
    created_at
) VALUES (
    NULL, -- Any source department (global rule)
    NULL, -- Any source location
    NULL, -- Target: NULL = escalate within same department hierarchy (authority lookup will resolve)
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
);

-- Note: No rules for L3 → L4 (max escalation level is L3)
-- The escalation worker will enforce this limit programmatically
--
-- Department-specific rules: To create department-specific escalation targets,
-- set to_department_id to the target department ID (e.g., escalate PWD → District Collector)
