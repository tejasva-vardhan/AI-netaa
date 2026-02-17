# Complaint Lifecycle Rules

This document explains the complaint lifecycle rules implemented in the system.

## Status Flow

### Initial Status
- When a complaint is created with all required fields (title, description, location_id), it starts with status `submitted`
- If required fields are missing, it starts with status `draft`
- The initial status is automatically recorded in `complaint_status_history`

### Status Transitions

The system enforces valid status transitions to maintain data integrity:

```
draft → submitted, draft
submitted → under_review, rejected, draft
under_review → in_progress, rejected, escalated
in_progress → resolved, rejected, escalated
resolved → closed
rejected → closed, under_review (can be reopened)
escalated → under_review, in_progress
closed → (terminal state - no further transitions)
```

**Implementation**: See `isValidStatusTransition()` in `service/complaint_service.go`

## Audit Requirements

### 1. Status History (complaint_status_history)

**Rule**: Every status change MUST create an entry in `complaint_status_history`

**When**: 
- Complaint creation (initial status)
- Any status update via `UpdateComplaintStatus()`

**What is recorded**:
- `old_status`: Previous status (NULL for initial creation)
- `new_status`: New status
- `changed_by_type`: Who made the change (user, officer, system, admin)
- `changed_by_user_id`: User ID if changed by user
- `changed_by_officer_id`: Officer ID if changed by officer
- `assigned_department_id`: Department assignment at time of change
- `assigned_officer_id`: Officer assignment at time of change
- `notes`: Optional notes about the change
- `created_at`: Timestamp (automatically set)

**Immutability**: This table is append-only. Records are never updated or deleted.

### 2. Audit Log (audit_log)

**Rule**: Every action MUST create an entry in `audit_log`

**When**:
- Complaint creation
- Status updates
- Any other critical operations

**What is recorded**:
- `entity_type`: Type of entity (e.g., "complaint")
- `entity_id`: ID of the entity
- `action`: Action performed (e.g., "create", "status_change")
- `action_by_type`: Who performed the action
- `action_by_user_id`: User ID if applicable
- `action_by_officer_id`: Officer ID if applicable
- `old_values`: JSON snapshot of previous state
- `new_values`: JSON snapshot of new state
- `changes`: JSON diff of what changed
- `ip_address`: Client IP address
- `user_agent`: Browser/device information
- `metadata`: Additional context (JSON)
- `created_at`: Timestamp

**Immutability**: This table is append-only. Records are never updated or deleted.

## Timestamp Management

### Automatic Timestamps

1. **resolved_at**: Automatically set when status changes to `resolved`
   - Only set if currently NULL (first time resolution)
   - Implementation: `UpdateComplaintStatus()` in repository

2. **closed_at**: Automatically set when status changes to `closed`
   - Only set if currently NULL (first time closure)
   - Implementation: `UpdateComplaintStatus()` in repository

### Manual Timestamps

- `created_at`: Set on record creation (database default)
- `updated_at`: Set on non-status updates (for non-audit changes)

## Assignment Tracking

### Department Assignment

- Can be set during complaint creation (optional)
- Can be updated during status changes
- Tracked in both `complaints` table and `complaint_status_history`

### Officer Assignment

- Can be set during complaint creation (optional)
- Can be updated during status changes
- Tracked in both `complaints` table and `complaint_status_history`

## Access Control

### Citizen View

- Users can only view their own complaints OR public complaints (`is_public = true`)
- Status timeline is visible to owners and for public complaints
- Attachments respect public visibility settings

### Internal Operations

- Status updates require officer/admin authentication
- Actor information (officer_id, user_id) is tracked in audit logs
- IP address and user agent are logged for security

## Error Handling

### Status Transition Errors

If an invalid status transition is attempted:
- Error returned: "invalid status transition from {old} to {new}"
- HTTP Status: 400 Bad Request
- No database changes are made

### Missing Audit Logs

- Audit log creation failures are logged but do not fail the operation
- This ensures system resilience (audit logging should not block core operations)
- In production, consider using async logging or retry mechanisms

## Implementation Details

### Service Layer (`service/complaint_service.go`)

1. **CreateComplaint()**:
   - Creates complaint record
   - Creates initial status history entry
   - Creates audit log entry
   - Links attachments

2. **UpdateComplaintStatus()**:
   - Validates status transition
   - Updates complaint status
   - Sets resolved_at/closed_at if applicable
   - Creates status history entry (REQUIRED)
   - Creates audit log entry (REQUIRED)

3. **GetStatusTimeline()**:
   - Retrieves all status history entries
   - Orders by created_at DESC (newest first)
   - Respects access control

### Repository Layer (`repository/complaint_repository.go`)

- All database operations
- Status history creation
- Audit log creation
- Attachment management

## Best Practices

1. **Always use transactions** for operations that modify multiple tables
   - Current implementation uses individual queries
   - In production, wrap status updates in transactions

2. **Validate before database operations**
   - Status transitions validated in service layer
   - Reduces database errors

3. **Log failures gracefully**
   - Audit log failures don't block operations
   - Consider retry mechanisms for critical audit logs

4. **Index optimization**
   - Status history queries use `(complaint_id, created_at DESC)` index
   - Audit log queries use multiple indexes for different access patterns

5. **Data consistency**
   - Status in `complaints.current_status` must match latest entry in `complaint_status_history`
   - Consider triggers or application-level checks

## Future Enhancements

1. **State Machine**: Implement a formal state machine library for status transitions
2. **Transaction Support**: Wrap status updates in database transactions
3. **Async Audit Logging**: Use message queue for audit log writes
4. **Validation Middleware**: Add request validation middleware
5. **Rate Limiting**: Add rate limiting for status updates
6. **Caching**: Cache complaint details for frequently accessed complaints
