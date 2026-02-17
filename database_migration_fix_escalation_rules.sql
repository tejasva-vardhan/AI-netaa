-- ============================================================================
-- Migration: Fix escalation_rules schema for pilot / global rules
-- ============================================================================
-- Purpose: Allow NULL on department columns so that:
--   - from_department_id = NULL  → rule applies to ANY department (global)
--   - to_department_id  = NULL  → escalate within same department hierarchy
--     (target department is resolved at runtime from complaint's assigned_department_id)
-- Authority-based design: escalation is by department + level, not by officer.
-- Safe to run on existing data (only column metadata changed).
-- ============================================================================

-- from_department_id: NULL = any source department (global rule)
-- (Most schemas already have this as NULL; ensure comment is clear.)
ALTER TABLE escalation_rules
MODIFY COLUMN from_department_id BIGINT NULL
COMMENT 'Source department filter. NULL = any department (global rule).';

-- to_department_id: NULL = same department hierarchy (dynamic authority resolution)
-- Required for pilot: rules must not hardcode department IDs.
ALTER TABLE escalation_rules
MODIFY COLUMN to_department_id BIGINT NULL
COMMENT 'Target department. NULL = escalate within same department hierarchy (resolved from complaint at runtime).';
