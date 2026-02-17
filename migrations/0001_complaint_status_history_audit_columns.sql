-- Idempotent migration: add actor_type, actor_id, reason to complaint_status_history if not present.
-- Safe to run multiple times. Prevents schema-code mismatch (e.g. escalation failures).

DELIMITER $$

DROP PROCEDURE IF EXISTS add_status_history_audit_columns$$

CREATE PROCEDURE add_status_history_audit_columns()
BEGIN
  DECLARE col_count INT DEFAULT 0;

  -- actor_type ENUM('user','authority','system')
  SELECT COUNT(*) INTO col_count
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'complaint_status_history'
    AND COLUMN_NAME = 'actor_type';
  IF col_count = 0 THEN
    ALTER TABLE complaint_status_history
      ADD COLUMN actor_type ENUM('user','authority','system') NULL
        COMMENT 'Who made the change: user, authority, or system';
  END IF;

  -- actor_id BIGINT
  SET col_count = 0;
  SELECT COUNT(*) INTO col_count
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'complaint_status_history'
    AND COLUMN_NAME = 'actor_id';
  IF col_count = 0 THEN
    ALTER TABLE complaint_status_history
      ADD COLUMN actor_id BIGINT NULL
        COMMENT 'User ID or officer ID of the actor (nullable for system)';
  END IF;

  -- reason TEXT
  SET col_count = 0;
  SELECT COUNT(*) INTO col_count
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'complaint_status_history'
    AND COLUMN_NAME = 'reason';
  IF col_count = 0 THEN
    ALTER TABLE complaint_status_history
      ADD COLUMN reason TEXT NULL
        COMMENT 'Reason for the status change when available';
  END IF;

END$$

DELIMITER ;

CALL add_status_history_audit_columns();
DROP PROCEDURE IF EXISTS add_status_history_audit_columns;
