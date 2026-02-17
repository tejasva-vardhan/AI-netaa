# Abuse Prevention Implementation

## Overview
Implements basic abuse-prevention checks for complaint submission: rate limiting, duplicate detection, and device fingerprinting.

## Requirements

1. **Rate Limiting**: Max 3 complaints per user per 24 hours
2. **Duplicate Detection**: Same user + same issue_summary + same pincode within 30 minutes → reject
3. **Device Fingerprint**: Hash of user_id + user_agent + screen_size stored in complaints table
4. **Error Messages**: Generic, non-accusatory messages

## Database Schema Changes

### Add `device_fingerprint` column to complaints table
```sql
ALTER TABLE complaints 
ADD COLUMN device_fingerprint VARCHAR(64) NULL 
COMMENT 'SHA256 hash of user_id + user_agent + screen_size';
```

### Add `pincode` column to complaints table (if not exists)
```sql
ALTER TABLE complaints 
ADD COLUMN pincode VARCHAR(10) NULL 
COMMENT 'Postal code / PIN code for complaint location';
```

### Indexes for performance
```sql
-- Index for rate limiting queries
CREATE INDEX idx_user_created_status ON complaints(user_id, created_at, current_status);

-- Index for duplicate detection queries
CREATE INDEX idx_user_title_pincode_created ON complaints(user_id, title, pincode, created_at);
```

## Implementation

### Files Created

1. **`service/abuse_prevention_service.go`**: Abuse prevention business logic
2. **`repository/abuse_prevention_repository.go`**: Database queries for abuse checks
3. **`database_abuse_prevention.sql`**: SQL schema changes

### 1. Abuse Prevention Service (`service/abuse_prevention_service.go`)

**Key Functions:**
- `CheckRateLimit(userID)`: Verifies user hasn't exceeded 3 complaints in 24 hours
- `CheckDuplicate(userID, issueSummary, pincode)`: Checks for duplicate within 30 minutes
- `GenerateDeviceFingerprint(userID, userAgent, screenSize)`: Creates SHA256 hash
- `ValidateComplaintSubmission(userID, issueSummary, pincode)`: Performs all checks

**Error Codes (Internal - NOT exposed to user):**
- `RATE_LIMIT_EXCEEDED`: User exceeded 3 complaints per 24 hours
- `DUPLICATE_SUBMISSION`: Duplicate complaint detected

### 2. Abuse Prevention Repository (`repository/abuse_prevention_repository.go`)

**Key Functions:**
- `CountComplaintsByUserInLast24Hours(userID)`: Counts complaints in last 24 hours
- `HasDuplicateComplaint(userID, issueSummary, pincode, withinDuration)`: Checks for duplicates

### 3. Integration Points

#### Update Complaint Creation Handler

**In `handler/complaint_handler.go`:**

The handler has been updated to:
1. Extract `X-Screen-Size` header from request
2. Call `abusePreventionService.ValidateComplaintSubmission()` before creating complaint
3. Generate device fingerprint using `service.GenerateDeviceFingerprint()`
4. Return `429 Too Many Requests` with generic message if abuse detected

**Key Changes:**
- Added `abusePreventionService` field to `ComplaintHandler`
- Added abuse check before complaint creation
- Device fingerprint generated and added to request

#### Update Complaint Service

**In `service/complaint_service.go`:**

The service has been updated to:
1. Store `pincode` from request in complaint entity
2. Store `device_fingerprint` from request in complaint entity

**Key Changes:**
- Added `Pincode` and `DeviceFingerprint` fields to `Complaint` model
- Service maps these fields from request to entity before database insert

#### Update Complaint Repository

**In `repository/complaint_repository.go`:**

The repository has been updated to:
1. Include `pincode` and `device_fingerprint` in INSERT statement
2. Pass these values in Exec() call

**Key Changes:**
- INSERT query includes `pincode` and `device_fingerprint` columns
- Exec() parameters include these fields

## SQL Queries

### Rate Limiting Query
```sql
SELECT COUNT(*)
FROM complaints
WHERE user_id = ?
  AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
  AND current_status != 'rejected';
```

### Duplicate Detection Query
```sql
SELECT COUNT(*) > 0
FROM complaints
WHERE user_id = ?
  AND title = ?
  AND pincode = ?
  AND created_at >= ?
  AND current_status != 'rejected';
```

## Error Responses

### Rate Limit Exceeded
```json
{
  "error": "Submission limit",
  "message": "You have reached the maximum number of complaints allowed per day. Please try again tomorrow.",
  "code": 429
}
```

### Duplicate Submission
```json
{
  "error": "Submission limit",
  "message": "A similar complaint was recently submitted. Please wait before submitting again.",
  "code": 429
}
```

**Note**: Error code `RATE_LIMIT_EXCEEDED` and `DUPLICATE_SUBMISSION` are internal only and NOT exposed to user.

## Frontend Integration

### Send Screen Size Header
Frontend must send screen size in request header when creating complaints:

```javascript
// In frontend/src/services/api.js
const screenSize = `${window.screen.width}x${window.screen.height}`;

// When creating complaint:
fetch('/api/v1/complaints', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
    'X-Screen-Size': screenSize, // REQUIRED: Add this header
  },
  body: JSON.stringify({
    title: '...',
    description: '...',
    pincode: '473551', // REQUIRED: Include pincode in request
    // ... other fields
  }),
});
```

**Note**: If `X-Screen-Size` header is missing, backend uses "unknown" as default.

## Testing Considerations

1. **Rate Limit Test**: Submit 3 complaints → 4th should be rejected
2. **Duplicate Test**: Submit same title + pincode within 30 minutes → should be rejected
3. **Device Fingerprint Test**: Verify hash is generated correctly and stored
4. **Time Window Test**: Submit duplicate after 30+ minutes → should be allowed
5. **Rejected Status Test**: Rejected complaints don't count toward rate limit

## Notes

- **No Account Blocking**: System only prevents submission, doesn't block user account
- **Generic Messages**: Error messages are user-friendly and non-accusatory
- **Status Filtering**: Only non-rejected complaints count toward limits
- **Device Fingerprint**: Used for analytics/pattern detection, not blocking
