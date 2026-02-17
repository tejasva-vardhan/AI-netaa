-- Migration: Complaint status history audit trail
-- Adds actor_type, actor_id, reason for full audit of who changed status and why.
-- Backward compatibility: existing rows keep NULL actor_type/actor_id/reason.

-- Add audit columns to complaint_status_history
ALTER TABLE complaint_status_history
  ADD COLUMN actor_type ENUM('system','authority','user') NULL
    COMMENT 'Who made the change: system, authority, or user'
    AFTER notes,
  ADD COLUMN actor_id BIGINT NULL
    COMMENT 'User ID or officer ID of the actor (nullable for system)'
    AFTER actor_type,
  ADD COLUMN reason TEXT NULL
    COMMENT 'Reason for the status change when available'
    AFTER actor_id;

-- Index for filtering by actor (optional, useful for audit queries)
CREATE INDEX idx_status_history_actor ON complaint_status_history (actor_type, actor_id);
