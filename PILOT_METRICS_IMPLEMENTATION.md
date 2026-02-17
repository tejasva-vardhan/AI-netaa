# Pilot Metrics & Logging Implementation

Backend-only metrics system for pilot evaluation and demos. No UI or analytics service integration required.

## Database Schema

**File:** `database_pilot_metrics.sql`

**Table:** `pilot_metrics_events`

```sql
CREATE TABLE IF NOT EXISTS pilot_metrics_events (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    event_type VARCHAR(50) NOT NULL,
    complaint_id BIGINT NULL,
    user_id BIGINT NULL,
    metadata JSON NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_event_type (event_type),
    INDEX idx_complaint_id (complaint_id),
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at),
    INDEX idx_event_type_created_at (event_type, created_at)
);
```

## Event Types

1. **`complaint_created`** - Complaint submission
2. **`first_authority_action`** - First authority status update
3. **`escalation_triggered`** - Escalation executed
4. **`complaint_resolved`** - Complaint resolved or closed
5. **`chat_abandoned`** - User abandons chat before submission

## Metrics Tracked

### 1. complaints_created_per_day
- **Event:** `complaint_created`
- **Query:** Count events by `event_type = 'complaint_created'` grouped by `DATE(created_at)`

### 2. time_to_first_authority_action
- **Event:** `first_authority_action`
- **Metadata:** `time_to_first_action_seconds`, `time_to_first_action_hours`
- **Calculation:** Time difference between complaint creation and first authority action

### 3. escalations_triggered_count
- **Event:** `escalation_triggered`
- **Query:** Count events by `event_type = 'escalation_triggered'`
- **Metadata:** `escalation_level`, `target_level`

### 4. complaints_resolved_count
- **Event:** `complaint_resolved`
- **Query:** Count events by `event_type = 'complaint_resolved'`
- **Metadata:** `time_to_resolution_seconds`, `time_to_resolution_hours`, `status`

### 5. chat_dropoffs
- **Event:** `chat_abandoned`
- **Query:** Count events by `event_type = 'chat_abandoned'`
- **Note:** Emitted when user resets/abandons chat without submitting

## Event Emission Locations

### 1. Complaint Submission
**File:** `service/complaint_service.go`  
**Method:** `CreateComplaint()`  
**Location:** After successful complaint creation (line ~179)  
**Event:** `complaint_created`  
**Metadata:**
- `complaint_number`
- `status`
- `priority`
- `has_category`
- `attachment_count`
- `assigned_department_id` (if assigned)

### 2. First Authority Action
**File:** `service/authority_service.go`  
**Method:** `UpdateComplaintStatus()`  
**Location:** After status history creation, checks if this is first authority action (line ~163)  
**Event:** `first_authority_action`  
**Detection:** Counts previous status history entries with `actor_type = 'authority'`  
**Metadata:**
- `time_to_first_action_seconds`
- `time_to_first_action_hours`
- `old_status`
- `new_status`
- `officer_id`

### 3. Escalation Triggered
**File:** `service/escalation_service.go`  
**Method:** `executeEscalation()`  
**Location:** After successful escalation record creation (line ~414)  
**Event:** `escalation_triggered`  
**Metadata:**
- `escalation_level` (current level before escalation)
- `target_level` (level escalated to)
- `from_department`
- `to_department`
- `reason`

### 4. Complaint Resolved
**File:** `service/authority_service.go`  
**Method:** `UpdateComplaintStatus()`  
**Location:** When status becomes `resolved` or `closed` (line ~196)  
**Event:** `complaint_resolved`  
**Metadata:**
- `time_to_resolution_seconds`
- `time_to_resolution_hours`
- `status` (resolved/closed)
- `old_status`
- `new_status`
- `officer_id`

### 5. Chat Abandoned
**File:** `handler/chat_handler.go`  
**Method:** `ResetChatDraft()`  
**Location:** When user resets chat draft (line ~30)  
**Event:** `chat_abandoned`  
**Metadata:**
- `action` (reset)

## Implementation Details

### Models
**File:** `models/entities.go`

- `PilotMetricsEventType` - Enum for event types
- `PilotMetricsEvent` - Struct for event records

### Repository
**File:** `repository/pilot_metrics_repository.go`

- `CreateEvent()` - Creates event with model
- `CreateEventWithMetadata()` - Creates event with metadata map (auto-serializes to JSON)

### Service
**File:** `service/pilot_metrics_service.go`

- `EmitComplaintCreated()` - Emits complaint_created event
- `EmitFirstAuthorityAction()` - Emits first_authority_action event (calculates time delta)
- `EmitEscalationTriggered()` - Emits escalation_triggered event
- `EmitComplaintResolved()` - Emits complaint_resolved event (calculates time delta)
- `EmitChatAbandoned()` - Emits chat_abandoned event

### Service Wiring
**File:** `main.go`

- `pilotMetricsRepo` initialized
- `pilotMetricsService` initialized
- Injected into:
  - `ComplaintService`
  - `AuthorityService`
  - `EscalationService`
  - `ChatHandler` (via routes)

## Query Examples

### Complaints Created Per Day
```sql
SELECT 
    DATE(created_at) as date,
    COUNT(*) as count
FROM pilot_metrics_events
WHERE event_type = 'complaint_created'
GROUP BY DATE(created_at)
ORDER BY date DESC;
```

### Average Time to First Authority Action
```sql
SELECT 
    AVG(JSON_EXTRACT(metadata, '$.time_to_first_action_hours')) as avg_hours
FROM pilot_metrics_events
WHERE event_type = 'first_authority_action';
```

### Escalations Triggered Count
```sql
SELECT COUNT(*) as total_escalations
FROM pilot_metrics_events
WHERE event_type = 'escalation_triggered';
```

### Complaints Resolved Count
```sql
SELECT COUNT(*) as total_resolved
FROM pilot_metrics_events
WHERE event_type = 'complaint_resolved';
```

### Chat Dropoffs Count
```sql
SELECT COUNT(*) as total_dropoffs
FROM pilot_metrics_events
WHERE event_type = 'chat_abandoned';
```

## Notes

- **Non-blocking:** All event emissions are non-blocking. Failures are logged but do not affect main flow.
- **Optional service:** Services check if `pilotMetricsService != nil` before emitting events.
- **Metadata:** All metadata is stored as JSON in the `metadata` column for flexibility.
- **Timestamps:** All events include `timestamp` in metadata (Unix timestamp).
- **Time calculations:** Time deltas (first action, resolution) are calculated at event emission time.
