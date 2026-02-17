# Escalation and Reminder Engine

## Overview

A rule-based escalation and reminder engine that periodically scans complaints and automatically escalates them or sends reminders based on configurable rules stored in the `escalation_rules` table.

## Architecture

### Components

1. **EscalationRepository** - Database operations for escalation rules and records
2. **EscalationService** - Core escalation logic and rule evaluation
3. **EscalationWorker** - Background worker that periodically processes escalations
4. **EscalationHandler** - HTTP endpoints for manual triggering

## How It Works

### 1. Periodic Scanning

The `EscalationWorker` runs in the background and periodically:
- Scans complaints by status and `updated_at` timestamp
- Evaluates escalation rules for each complaint
- Creates escalation records when conditions are met
- Updates complaint status via `complaint_status_history`
- Logs all actions to `audit_log`

### 2. Rule Evaluation

Rules are stored in `escalation_rules` table with JSON `conditions`:

```json
{
  "statuses": ["verified", "under_review"],
  "time_based": {
    "hours_since_last_update": 48,
    "hours_since_status_change": 24
  },
  "priorities": ["high", "urgent"],
  "is_reminder": false
}
```

### 3. Escalation Flow

```
Complaint in status: verified/under_review/in_progress
    ↓
Check last update time
    ↓
Load escalation rules (matching department/location)
    ↓
Evaluate conditions (status, time, priority)
    ↓
Check idempotency (not already escalated at this level)
    ↓
Create escalation record
    ↓
Update status to "escalated" via status_history
    ↓
Log to audit_log
```

## Idempotency

The engine is **idempotent** - safe to run multiple times:

1. **Escalation Level Check**: Checks if complaint already escalated at the same level
2. **Time Window**: Only escalates if no escalation in last hour (configurable)
3. **Status Check**: Only processes complaints in specific statuses
4. **Rule Matching**: Only applies rules matching department/location

### Idempotency Mechanisms

```go
// Check if already escalated at this level recently
alreadyEscalated, err := s.escalationRepo.HasExistingEscalation(
    complaintID,
    escalationLevel,
    1, // Within 1 hour
)
```

## Escalation Rules

### Rule Structure

Rules are stored in `escalation_rules` table:

| Field | Description |
|-------|-------------|
| `from_department_id` | Source department (NULL = any) |
| `from_location_id` | Source location (NULL = any) |
| `to_department_id` | Target department (required) |
| `to_location_id` | Target location (NULL = same) |
| `escalation_level` | Level in hierarchy (1, 2, 3...) |
| `conditions` | JSON conditions |
| `is_active` | Active status |

### Conditions JSON Format

```json
{
  "statuses": ["verified", "under_review"],
  "time_based": {
    "hours_since_last_update": 48,
    "hours_since_status_change": 24,
    "hours_since_creation": 72
  },
  "priorities": ["high", "urgent"],
  "is_reminder": false,
  "reminder_interval_hours": 24
}
```

### Condition Types

1. **Status Conditions**: Escalate if complaint is in these statuses
2. **Time-Based Conditions**:
   - `hours_since_last_update`: Time since `updated_at`
   - `hours_since_status_change`: Time since last status change
   - `hours_since_creation`: Time since complaint creation
3. **Priority Conditions**: Escalate based on priority level
4. **Reminder Conditions**: Send reminder instead of escalation

## Reminders

Reminders are logged to `audit_log` but **do not escalate** the complaint:

- `action = "reminder"`
- Sent based on `reminder_interval_hours`
- Prevents duplicate reminders within interval
- Useful for notifying without changing status

### Reminder Flow

```
Complaint matches reminder conditions
    ↓
Check last reminder time
    ↓
If interval passed → Log reminder to audit_log
    ↓
Do NOT escalate (status unchanged)
```

## Escalation Process

### Step 1: Find Applicable Rules

```go
// Get rules matching complaint's department and location
for _, rule := range rules {
    if rule.EscalationLevel == nextLevel {
        if ruleMatchesComplaint(rule, candidate) {
            applicableRules = append(applicableRules, rule)
        }
    }
}
```

### Step 2: Evaluate Conditions

```go
// Check status
if len(conditions.Statuses) > 0 {
    // Must match one of the statuses
}

// Check time-based conditions
if conditions.TimeBased.HoursSinceLastUpdate > 0 {
    // Must have passed X hours since last update
}

// Check priority
if len(conditions.Priorities) > 0 {
    // Must match one of the priorities
}
```

### Step 3: Create Escalation Record

```go
escalation := &models.ComplaintEscalation{
    ComplaintID:     complaintID,
    FromDepartmentID: currentDepartmentID,
    ToDepartmentID:  rule.ToDepartmentID,
    EscalationLevel: rule.EscalationLevel,
    EscalatedByType: models.ActorSystem,
    Reason:          "Auto-escalated after 48 hours",
}
```

### Step 4: Update Status

```go
// Update via status history (REQUIRED)
statusHistory := &models.ComplaintStatusHistory{
    ComplaintID:   complaintID,
    OldStatus:    currentStatus,
    NewStatus:    models.StatusEscalated,
    ChangedByType: models.ActorSystem,
    Notes:         "Escalated to level 2",
}
```

### Step 5: Log to Audit

```go
auditLog := &models.AuditLog{
    EntityType:   "complaint",
    EntityID:     complaintID,
    Action:       "escalation",
    ActionByType: models.ActorSystem,
    Metadata:     escalationMetadata,
}
```

## Background Worker

### Configuration

```go
worker := worker.NewEscalationWorker(
    escalationService,
    1*time.Hour, // Process every hour
)
worker.Start()
```

### Worker Lifecycle

1. **Start**: Begins periodic processing
2. **Process**: Scans complaints and processes escalations
3. **Stop**: Gracefully stops processing

### Processing Interval

Default: **1 hour**

Can be configured:
```go
worker := worker.NewEscalationWorker(
    escalationService,
    30*time.Minute, // Process every 30 minutes
)
```

## API Endpoints

### POST `/api/v1/escalations/process`

Manually trigger escalation processing (useful for testing).

**Request**: None

**Response**:
```json
{
  "processed": 5,
  "results": [
    {
      "complaint_id": 123,
      "escalated": true,
      "escalation_id": 1,
      "new_status": "escalated",
      "reason": "Auto-escalated after 48 hours",
      "processed_at": "2026-02-12T10:00:00Z"
    }
  ]
}
```

## Example Escalation Rules

### Rule 1: Escalate after 48 hours

```sql
INSERT INTO escalation_rules (
    from_department_id, to_department_id, escalation_level, conditions, is_active
) VALUES (
    NULL, -- Any department
    2,    -- Escalate to department 2
    1,    -- Level 1
    '{
        "statuses": ["verified", "under_review"],
        "time_based": {
            "hours_since_last_update": 48
        }
    }',
    true
);
```

### Rule 2: Reminder after 24 hours

```sql
INSERT INTO escalation_rules (
    from_department_id, to_department_id, escalation_level, conditions, is_active
) VALUES (
    1,    -- Department 1
    1,    -- Same department (reminder only)
    0,    -- Level 0 (reminder)
    '{
        "statuses": ["under_review"],
        "time_based": {
            "hours_since_status_change": 24
        },
        "is_reminder": true,
        "reminder_interval_hours": 24
    }',
    true
);
```

### Rule 3: Escalate high priority after 24 hours

```sql
INSERT INTO escalation_rules (
    from_department_id, to_department_id, escalation_level, conditions, is_active
) VALUES (
    NULL, -- Any department
    3,    -- Escalate to department 3
    2,    -- Level 2
    '{
        "statuses": ["verified", "under_review", "in_progress"],
        "priorities": ["high", "urgent"],
        "time_based": {
            "hours_since_last_update": 24
        }
    }',
    true
);
```

## Audit Logging

### Escalation Log Entry

```json
{
  "entity_type": "complaint",
  "entity_id": 123,
  "action": "escalation",
  "action_by_type": "system",
  "metadata": {
    "escalation_id": 1,
    "escalation_level": 2,
    "from_department": 1,
    "to_department": 2,
    "reason": "Auto-escalated after 48 hours"
  }
}
```

### Reminder Log Entry

```json
{
  "entity_type": "complaint",
  "entity_id": 123,
  "action": "reminder",
  "action_by_type": "system",
  "metadata": {
    "reminder_reason": "Reminder sent (last reminder 24.5 hours ago)",
    "rule_id": 5
  }
}
```

## Status Transitions

### Escalation Status Flow

```
verified → escalated (via escalation engine)
under_review → escalated (via escalation engine)
in_progress → escalated (via escalation engine)
escalated → under_review (manual review)
escalated → in_progress (work started)
```

## Configuration-Driven Design

### No Hard-Coded Values

- **No hard-coded days**: All time thresholds in `conditions` JSON
- **No hard-coded hierarchy**: Escalation levels defined in rules
- **No hard-coded departments**: Department mapping in rules
- **No hard-coded statuses**: Status conditions in JSON

### Example: Flexible Configuration

```json
{
  "statuses": ["verified", "under_review"],
  "time_based": {
    "hours_since_last_update": 48,
    "hours_since_status_change": 24
  },
  "priorities": ["high", "urgent"]
}
```

## Error Handling

- **Rule parsing errors**: Skip rule, continue processing
- **Database errors**: Log error, continue with next complaint
- **Audit log errors**: Log error but don't fail escalation (resilient)
- **Status history errors**: Fail escalation (critical)

## Performance Considerations

1. **Batch Processing**: Processes all candidates in one run
2. **Indexing**: Ensure indexes on:
   - `complaints(current_status, updated_at)`
   - `complaint_escalations(complaint_id, escalation_level, created_at)`
   - `escalation_rules(is_active, escalation_level)`
3. **Query Optimization**: Uses efficient queries with proper WHERE clauses
4. **Worker Interval**: Configurable based on load

## Testing

### Manual Trigger

```bash
curl -X POST http://localhost:8080/api/v1/escalations/process
```

### Unit Testing

```go
// Test escalation service
results, err := escalationService.ProcessEscalations()
assert.NoError(t, err)
assert.Greater(t, len(results), 0)
```

## Future Enhancements

1. **Email Notifications**: Send emails when escalations occur
2. **SMS Notifications**: Send SMS for urgent escalations
3. **Webhook Support**: Trigger webhooks on escalation
4. **Escalation Templates**: Pre-defined escalation templates
5. **Analytics**: Track escalation patterns and effectiveness
6. **Dynamic Rules**: Update rules without code changes
