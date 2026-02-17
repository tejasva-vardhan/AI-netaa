-- Add 'verified' to complaint status ENUMs (required for admin verification flow).
-- Idempotent: safe to run on DBs that already have 'verified' (manual add).

-- complaints.current_status
ALTER TABLE complaints
  MODIFY COLUMN current_status ENUM('draft','submitted','verified','under_review','in_progress','resolved','rejected','closed','escalated') NOT NULL DEFAULT 'draft';

-- complaint_status_history.old_status, new_status
ALTER TABLE complaint_status_history
  MODIFY COLUMN old_status ENUM('draft','submitted','verified','under_review','in_progress','resolved','rejected','closed','escalated') NULL;
ALTER TABLE complaint_status_history
  MODIFY COLUMN new_status ENUM('draft','submitted','verified','under_review','in_progress','resolved','rejected','closed','escalated') NOT NULL;
