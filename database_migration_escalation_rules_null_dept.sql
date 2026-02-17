-- Migration: Allow NULL for to_department_id in escalation_rules
-- Purpose: Support global escalation rules (NULL = escalate within same department hierarchy)
-- Date: 2026-02-12

-- Allow NULL for to_department_id (was NOT NULL)
ALTER TABLE escalation_rules
MODIFY COLUMN to_department_id BIGINT NULL COMMENT 'Target department (NULL = escalate within same department hierarchy)';
