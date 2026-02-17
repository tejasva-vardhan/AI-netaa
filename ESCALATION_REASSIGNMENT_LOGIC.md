# Escalation Reassignment Logic - Pseudocode

## Overview
Extends escalation logic to reassign complaints to real **authorities** (not officers) based on department + pincode + escalation level.

**Key Principles:**
- Escalation is an **event**, NOT a status change
- Complaints maintain their lifecycle status (submitted → under_review → in_progress → resolved → closed)
- Escalation reassigns to a new **authority** (higher level in hierarchy)
- Officers work within authorities; escalation changes the authority, not individual officers

## Database Schema Extensions

### Required Schema Changes

#### 1. Add `current_escalation_level` to complaints table
```sql
ALTER TABLE complaints 
ADD COLUMN current_escalation_level INT NOT NULL DEFAULT 0 
COMMENT 'Current escalation level (0 = L1, 1 = L2, 2 = L3, etc.)';
```

#### 2. Add `pincode` column to complaints table
```sql
ALTER TABLE complaints 
ADD COLUMN pincode VARCHAR(10) NULL 
COMMENT 'Postal code / PIN code for complaint location';
```

#### 3. Replace `assigned_officer_id` with `assigned_authority_id`
```sql
-- If assigned_officer_id exists, migrate data first, then:
ALTER TABLE complaints 
DROP COLUMN assigned_officer_id,
ADD COLUMN assigned_authority_id BIGINT NULL 
COMMENT 'Currently assigned authority (references authorities table)';

-- Add index for performance
CREATE INDEX idx_assigned_authority ON complaints(assigned_authority_id);
```

**Note**: Assumes `authorities` table exists (separate from `officers` table).

#### 4. Update complaint_status_history table
```sql
-- Replace assigned_officer_id with assigned_authority_id
ALTER TABLE complaint_status_history
DROP COLUMN assigned_officer_id,
ADD COLUMN assigned_authority_id BIGINT NULL 
COMMENT 'Authority assigned at time of status change';
```

#### 5. Update complaint_escalations table
```sql
-- Replace from_officer_id and to_officer_id with authority fields
ALTER TABLE complaint_escalations
DROP COLUMN from_officer_id,
DROP COLUMN to_officer_id,
ADD COLUMN from_authority_id BIGINT NULL 
COMMENT 'Authority escalated from',
ADD COLUMN to_authority_id BIGINT NULL 
COMMENT 'Authority escalated to';
```

---

## Core Logic: `executeEscalationWithReassignment`

```go
// executeEscalationWithReassignment performs escalation and reassigns to next authority
func (s *EscalationService) executeEscalationWithReassignment(
    candidate models.EscalationCandidate,
    rule models.EscalationRule,
    reason string,
) (*models.EscalationResult, error) {
    
    // ============================================================
    // STEP 1: DETERMINE NEXT ESCALATION LEVEL
    // ============================================================
    
    // Get current escalation level from complaint
    currentLevel, err := s.getCurrentEscalationLevel(candidate.ComplaintID)
    if err != nil {
        return nil, fmt.Errorf("failed to get current escalation level: %w", err)
    }
    
    // Calculate next level
    nextLevel := currentLevel + 1
    
    // Validate: Don't escalate beyond max level (e.g., L4 = level 3)
    maxLevel := 3 // L1=0, L2=1, L3=2, L4=3
    if nextLevel > maxLevel {
        return nil, fmt.Errorf("complaint already at max escalation level %d", currentLevel)
    }
    
    // ============================================================
    // STEP 2: GET COMPLAINT PINCODE (DIRECT FROM COMPLAINT)
    // ============================================================
    
    // Get complaint to read pincode directly
    complaint, err := s.complaintRepo.GetComplaintByID(candidate.ComplaintID)
    if err != nil {
        return nil, fmt.Errorf("failed to get complaint: %w", err)
    }
    
    // Read pincode directly from complaints.pincode column
    if !complaint.Pincode.Valid || complaint.Pincode.String == "" {
        return nil, fmt.Errorf("complaint %d missing pincode", candidate.ComplaintID)
    }
    pincode := complaint.Pincode.String
    
    // ============================================================
    // STEP 3: FIND NEXT AUTHORITY BY DEPARTMENT + PINCODE + LEVEL
    // ============================================================
    
    // Determine target department (from escalation rule)
    targetDepartmentID := rule.ToDepartmentID
    
    // Find authority (NOT officer) matching:
    //   - department_id = targetDepartmentID
    //   - pincode coverage matches complaint pincode
    //   - escalation_level = nextLevel
    
    nextAuthority, err := s.findAuthorityByDepartmentPincodeLevel(
        targetDepartmentID,
        pincode,
        nextLevel,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to find next authority: %w", err)
    }
    
    if nextAuthority == nil {
        return nil, fmt.Errorf("no authority found for department %d, pincode %s, level %d",
            targetDepartmentID, pincode, nextLevel)
    }
    
    // ============================================================
    // STEP 4: IDEMPOTENCY CHECK (CRITICAL)
    // ============================================================
    
    // Check if already escalated to this level recently (within 1 hour)
    alreadyEscalated, err := s.escalationRepo.HasExistingEscalation(
        candidate.ComplaintID,
        nextLevel,
        1, // within 1 hour
    )
    if err != nil {
        return nil, fmt.Errorf("idempotency check failed: %w", err)
    }
    if alreadyEscalated {
        return nil, nil // Already escalated, skip (idempotent)
    }
    
    // ============================================================
    // STEP 5: TRANSACTION - UPDATE COMPLAINT & CREATE HISTORY
    // ============================================================
    
    // Start database transaction
    tx, err := s.complaintRepo.BeginTransaction()
    if err != nil {
        return nil, fmt.Errorf("failed to start transaction: %w", err)
    }
    defer tx.Rollback() // Rollback on error
    
    // Step 5a: Update complaint assignment and escalation level
    // IMPORTANT: Do NOT change current_status - escalation is an event, not a status
    // Preserve existing status (e.g., under_review, in_progress)
    err = s.complaintRepo.UpdateComplaintAssignmentAndLevel(
        tx,
        candidate.ComplaintID,
        targetDepartmentID,           // assigned_department_id
        nextAuthority.AuthorityID,     // assigned_authority_id (NOT officer_id)
        nextLevel,                    // current_escalation_level
        // DO NOT change current_status - keep existing status
    )
    if err != nil {
        return nil, fmt.Errorf("failed to update complaint: %w", err)
    }
    
    // Step 5b: Create status history entry (REQUIRED)
    // IMPORTANT: Escalation is logged as an event, NOT a status change
    // Status remains the same (e.g., under_review stays under_review)
    statusHistory := &models.ComplaintStatusHistory{
        ComplaintID:   candidate.ComplaintID,
        OldStatus:     sql.NullString{String: string(candidate.CurrentStatus), Valid: true},
        NewStatus:     candidate.CurrentStatus, // SAME STATUS - escalation is event, not status change
        ChangedByType: models.ActorSystem,
        Notes: sql.NullString{
            String: fmt.Sprintf(
                "Escalation event: Level %d → Level %d. Reason: %s",
                currentLevel,
                nextLevel,
                reason,
            ),
            Valid: true,
        },
        AssignedDepartmentID: sql.NullInt64{Int64: targetDepartmentID, Valid: true},
        AssignedAuthorityID:  sql.NullInt64{Int64: nextAuthority.AuthorityID, Valid: true},
        // Metadata stored in audit_log, not here
    }
    
    historyID, err := s.complaintRepo.CreateStatusHistoryInTransaction(tx, statusHistory)
    if err != nil {
        return nil, fmt.Errorf("failed to create status history: %w", err)
    }
    
    // Step 5c: Create escalation record
    escalation := &models.ComplaintEscalation{
        ComplaintID:        candidate.ComplaintID,
        FromDepartmentID:   candidate.AssignedDepartmentID,
        FromAuthorityID:    candidate.AssignedAuthorityID, // From authority (not officer)
        ToDepartmentID:     targetDepartmentID,
        ToAuthorityID:      sql.NullInt64{Int64: nextAuthority.AuthorityID, Valid: true}, // To authority
        EscalationLevel:    nextLevel,
        EscalatedByType:    models.ActorSystem,
        Reason:             sql.NullString{String: reason, Valid: true},
        StatusHistoryID:   sql.NullInt64{Int64: historyID, Valid: true},
    }
    
    err = s.escalationRepo.CreateEscalationInTransaction(tx, escalation)
    if err != nil {
        return nil, fmt.Errorf("failed to create escalation record: %w", err)
    }
    
    // Step 5d: Create audit log entry with escalation metadata
    auditMetadata := map[string]interface{}{
        "escalation_id":      escalation.EscalationID,
        "event_type":         "escalation", // Explicit event type
        "from_level":         currentLevel,
        "to_level":           nextLevel,
        "from_department_id": candidate.AssignedDepartmentID,
        "to_department_id":   targetDepartmentID,
        "from_authority_id":  candidate.AssignedAuthorityID,
        "to_authority_id":    nextAuthority.AuthorityID,
        "pincode":            pincode,
        "reason":             reason, // e.g., "SLA breach"
        "status_preserved":   string(candidate.CurrentStatus), // Status did not change
    }
    
    err = s.logEscalationActionInTransaction(tx, candidate.ComplaintID, "escalation", auditMetadata)
    if err != nil {
        // Log error but don't fail - audit logging should be resilient
    }
    
    // Commit transaction
    err = tx.Commit()
    if err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    // ============================================================
    // STEP 6: RETURN RESULT
    // ============================================================
    
    return &models.EscalationResult{
        ComplaintID:  candidate.ComplaintID,
        Escalated:    true,
        EscalationID: &escalation.EscalationID,
        NewStatus:    stringPtr(string(candidate.CurrentStatus)), // Status unchanged
        Reason:       fmt.Sprintf("Escalated from L%d to L%d: %s", currentLevel+1, nextLevel+1, reason),
        ProcessedAt:  time.Now(),
    }, nil
}
```

---

## Helper Functions

### `getCurrentEscalationLevel`
```go
// getCurrentEscalationLevel gets current escalation level from complaint
func (s *EscalationService) getCurrentEscalationLevel(complaintID int64) (int, error) {
    // Option 1: Read from complaints.current_escalation_level (if field exists)
    complaint, err := s.complaintRepo.GetComplaintByID(complaintID)
    if err != nil {
        return 0, err
    }
    // If Complaint struct has CurrentEscalationLevel field:
    // return complaint.CurrentEscalationLevel, nil
    
    // Option 2: Derive from complaint_escalations table
    return s.escalationRepo.GetLastEscalationLevel(complaintID)
}
```

### `getComplaintPincode` (REMOVED - Use direct column access)
```go
// REMOVED: extractPincodeFromLocation
// Pincode is now stored directly in complaints.pincode column
// Access via: complaint.Pincode.String
```

### `findAuthorityByDepartmentPincodeLevel`
```go
// findAuthorityByDepartmentPincodeLevel finds AUTHORITY (not officer) matching department + pincode + level
func (s *EscalationService) findAuthorityByDepartmentPincodeLevel(
    departmentID int64,
    pincode string,
    escalationLevel int,
) (*models.Authority, error) {
    
    // Query authorities table (assumes it exists with structure):
    //   authority_id, department_id, pincode_coverage (JSON array or separate table),
    //   escalation_level, is_active
    
    // Strategy 1: Direct query if authorities table has pincode_coverage column
    query1 := `
        SELECT authority_id, name, department_id, escalation_level, is_active
        FROM authorities
        WHERE department_id = ?
          AND escalation_level = ?
          AND is_active = true
          AND (
              -- Direct pincode match
              pincode_coverage LIKE ?
              -- OR pincode_coverage is JSON array containing pincode
              OR JSON_CONTAINS(pincode_coverage, ?)
          )
        LIMIT 1
    `
    
    // Strategy 2: If pincode coverage is in separate table (authority_pincode_coverage)
    query2 := `
        SELECT a.authority_id, a.name, a.department_id, a.escalation_level, a.is_active
        FROM authorities a
        JOIN authority_pincode_coverage apc ON a.authority_id = apc.authority_id
        WHERE a.department_id = ?
          AND a.escalation_level = ?
          AND a.is_active = true
          AND apc.pincode = ?
        LIMIT 1
    `
    
    var authority models.Authority
    err := s.db.QueryRow(query2, departmentID, escalationLevel, pincode).Scan(
        &authority.AuthorityID,
        &authority.Name,
        &authority.DepartmentID,
        &authority.EscalationLevel,
        &authority.IsActive,
    )
    
    if err == sql.ErrNoRows {
        return nil, nil // No authority found (not an error)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to find authority: %w", err)
    }
    
    return &authority, nil
}
```

### `UpdateComplaintAssignmentAndLevel` (Repository Method)
```go
// UpdateComplaintAssignmentAndLevel updates complaint assignment and escalation level
// IMPORTANT: Does NOT change current_status - escalation is an event, not status change
func (r *ComplaintRepository) UpdateComplaintAssignmentAndLevel(
    tx *sql.Tx,
    complaintID int64,
    departmentID int64,
    authorityID int64, // assigned_authority_id (NOT officer_id)
    escalationLevel int,
) error {
    query := `
        UPDATE complaints
        SET assigned_department_id = ?,
            assigned_authority_id = ?,
            current_escalation_level = ?,
            updated_at = NOW()
            -- DO NOT update current_status - escalation preserves status
        WHERE complaint_id = ?
    `
    
    _, err := tx.Exec(
        query,
        departmentID,
        authorityID,
        escalationLevel,
        complaintID,
    )
    if err != nil {
        return fmt.Errorf("failed to update complaint: %w", err)
    }
    
    return nil
}
```

---

## Transaction Safety

### BeginTransaction (Repository)
```go
// BeginTransaction starts a new database transaction
func (r *ComplaintRepository) BeginTransaction() (*sql.Tx, error) {
    return r.db.Begin()
}
```

### CreateStatusHistoryInTransaction
```go
// CreateStatusHistoryInTransaction creates status history within transaction
// For escalations: old_status = new_status (status preserved)
func (r *ComplaintRepository) CreateStatusHistoryInTransaction(
    tx *sql.Tx,
    history *models.ComplaintStatusHistory,
) (int64, error) {
    query := `
        INSERT INTO complaint_status_history (
            complaint_id, old_status, new_status, changed_by_type,
            changed_by_user_id, changed_by_officer_id,
            assigned_department_id, assigned_authority_id, notes
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
    
    result, err := tx.Exec(
        query,
        history.ComplaintID,
        history.OldStatus,
        history.NewStatus, // For escalation: same as old_status
        history.ChangedByType,
        history.ChangedByUserID,
        history.ChangedByOfficerID,
        history.AssignedDepartmentID,
        history.AssignedAuthorityID, // assigned_authority_id (NOT officer_id)
        history.Notes,
    )
    if err != nil {
        return 0, fmt.Errorf("failed to create status history: %w", err)
    }
    
    historyID, err := result.LastInsertId()
    if err != nil {
        return 0, fmt.Errorf("failed to get history ID: %w", err)
    }
    
    return historyID, nil
}
```

---

## Idempotency Guarantees

1. **Check Before Escalation**: `HasExistingEscalation()` checks if escalation at same level exists within time window
2. **Transaction Isolation**: Database transaction ensures atomicity
3. **Unique Constraint**: Consider adding unique constraint on `(complaint_id, escalation_level)` in `complaint_escalations` table (if business logic allows)

---

## Status History Metadata

### Status History Entry (complaint_status_history)
- **old_status**: Previous status (e.g., "under_review")
- **new_status**: **SAME as old_status** (status preserved - escalation is event, not status change)
- **notes**: Contains escalation event description:
  ```
  "Escalation event: Level {currentLevel} → Level {nextLevel}. Reason: {reason}"
  ```
- **assigned_authority_id**: New authority assigned (NOT officer_id)

### Audit Log Metadata (audit_log.metadata JSON)
```json
{
  "escalation_id": 123,
  "event_type": "escalation",
  "from_level": 0,
  "to_level": 1,
  "from_department_id": 1,
  "to_department_id": 1,
  "from_authority_id": 5,
  "to_authority_id": 10,
  "pincode": "473551",
  "reason": "SLA breach",
  "status_preserved": "under_review"
}
```

**Key Points:**
- Status history shows escalation as an event (old_status = new_status)
- Detailed escalation metadata stored in audit_log
- Reason typically: "SLA breach" or specific breach description

---

## Integration Points

### Update `executeEscalation` in `escalation_service.go`
Replace existing `executeEscalation` method with `executeEscalationWithReassignment`:

```go
// OLD:
return s.executeEscalation(candidate, rule, reason)

// NEW:
return s.executeEscalationWithReassignment(candidate, rule, reason)
```

---

## Testing Considerations

1. **Test idempotency**: Run escalation twice for same complaint → should skip second time
2. **Test level progression**: L1 → L2 → L3 → L4
3. **Test authority lookup**: Verify correct officer selected by department + pincode + level
4. **Test transaction rollback**: Simulate error mid-transaction → verify no partial updates
5. **Test missing authority**: No officer found → should return clear error
6. **Test pincode extraction**: Various location hierarchies → correct pincode extracted

---

## Notes

- **Authorities vs Officers**: Escalation reassigns to **authorities** (higher-level entities), not individual officers
- **Status Preservation**: Escalation does NOT change complaint status - it's an event logged in history
- **Pincode Direct Access**: Complaints table has explicit `pincode` column - no extraction from location needed
- **No Hardcoded IDs**: All authority lookups use department_id + pincode + level
- **Deterministic**: Same inputs always produce same escalation path
- **Auditable**: Every escalation creates status history (event) and audit log entry
- **Transaction-Safe**: All updates happen atomically

## Lifecycle Status Flow (Unchanged by Escalation)

```
submitted → under_review → in_progress → resolved → closed
```

**Escalation can occur at any stage** (under_review, in_progress) but does NOT change the status.
The complaint continues its lifecycle while being reassigned to a higher authority.
