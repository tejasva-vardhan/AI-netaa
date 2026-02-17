# Verification Engine Implementation Summary

## Overview

A rule-based complaint verification engine has been implemented that automatically verifies complaints before they can proceed to review. The system ensures data quality, prevents duplicates, and maintains complete audit trails.

## Files Created/Modified

### New Files

1. **`models/verification.go`**
   - Verification reason codes
   - Verification request/response DTOs
   - Verification configuration
   - User model for verification checks

2. **`repository/verification_repository.go`**
   - `IsUserPhoneVerified()` - Checks phone verification status
   - `HasLiveCaptureAttachment()` - Checks for live capture attachments
   - `FindDuplicateComplaints()` - Finds duplicate complaints using Haversine formula
   - `IncrementSupporterCount()` - Increments supporter count for merged duplicates
   - `AddSupporter()` - Adds user as supporter to original complaint
   - `GetComplaintCoordinates()` - Retrieves complaint coordinates
   - `GetComplaintCategory()` - Retrieves complaint category

3. **`service/verification_service.go`**
   - `VerifyComplaint()` - Main verification logic implementing all rules
   - `createVerificationResult()` - Creates result and logs to audit
   - `logVerificationDecision()` - Logs all verification decisions to audit_log

4. **`handler/verification_handler.go`**
   - `VerifyComplaint()` - HTTP handler for verification endpoint

5. **`VERIFICATION_RULES.md`**
   - Complete documentation of verification rules
   - API examples
   - Configuration options
   - Testing examples

### Modified Files

1. **`models/entities.go`**
   - Added `StatusVerified` to ComplaintStatus enum

2. **`routes/routes.go`**
   - Added POST `/api/v1/complaints/{id}/verify` endpoint
   - Updated SetupRoutes to accept verification service

3. **`service/complaint_service.go`**
   - Updated `isValidStatusTransition()` to include `verified` status transitions

4. **`main.go`**
   - Initialized verification repository and service
   - Passed verification service to routes

## Verification Rules Implementation

### Rule 1: Live Capture Attachment ✅
- **Check**: At least one attachment exists
- **Note**: Schema doesn't have `live_capture` field, so currently checks for any attachment
- **Future**: Add `live_capture` boolean field to `complaint_attachments` table

### Rule 2: GPS Accuracy ✅
- **Check**: GPS accuracy ≤ threshold (default: 100 meters)
- **Implementation**: Optional field in verification request
- **Configurable**: Via `VerificationConfig.GPSAccuracyThreshold`

### Rule 3: Phone Verification ✅
- **Check**: `users.phone_verified_at IS NOT NULL`
- **Implementation**: Direct database query
- **No schema changes needed**

### Rule 4: Duplicate Detection ✅
- **Check**: Same category + location within radius + within time window
- **Algorithm**: Haversine formula for distance calculation
- **Merging**: Increments supporter_count, adds supporter record, logs audit
- **Configurable**: Radius and time window via `VerificationConfig`

## Duplicate Merging Process

When a duplicate is found:

1. **Find Original**: Identifies oldest complaint with matching criteria
2. **Increment Count**: Updates `supporter_count` on original complaint
3. **Add Supporter**: Creates entry in `complaint_supporters` table:
   - `is_duplicate = true`
   - `duplicate_notes = "Merged from complaint #<id>"`
4. **Log Audit**: Creates audit log entry for both complaints
5. **Return Result**: Returns `duplicate_complaint_id` and updated `supporter_count`

**Important**: The new complaint remains in "submitted" status (not verified) and is not deleted.

## Status Transitions

### New Transitions Added

- `submitted` → `verified` (automatic via verification)
- `verified` → `under_review` (manual review)
- `verified` → `in_progress` (work started)
- `verified` → `rejected` (if invalid)

## Audit Logging

**Every verification decision is logged** with:
- `action = "verification"`
- `action_by_type = "system"`
- `metadata` containing:
  - Verification result (verified: true/false)
  - Reason code
  - Reason message
  - Rules passed/failed
  - GPS accuracy (if provided)
  - Duplicate complaint ID (if duplicate found)

## API Endpoint

### POST `/api/v1/complaints/{id}/verify`

**Request**:
```json
{
  "gps_accuracy": 45.5  // Optional
}
```

**Response (Success)**:
```json
{
  "complaint_id": 123,
  "verified": true,
  "reason_code": "VERIFIED",
  "reason_message": "Complaint verified successfully"
}
```

**Response (Duplicate Found)**:
```json
{
  "complaint_id": 123,
  "verified": false,
  "reason_code": "DUPLICATE_FOUND",
  "reason_message": "Duplicate complaint found. Merged with complaint #100",
  "duplicate_complaint_id": 100,
  "supporter_count": 3
}
```

## Reason Codes

| Code | Description |
|------|-------------|
| `VERIFIED` | All rules passed |
| `NO_LIVE_CAPTURE` | No live capture attachment found |
| `GPS_ACCURACY_EXCEEDED` | GPS accuracy exceeds threshold |
| `PHONE_NOT_VERIFIED` | User phone not verified |
| `DUPLICATE_FOUND` | Duplicate complaint found and merged |

## Configuration

Default configuration:
```go
config := &models.VerificationConfig{
    GPSAccuracyThreshold:         100.0,              // meters
    DuplicateDetectionRadius:     50.0,              // meters
    DuplicateDetectionTimeWindow: 24 * time.Hour,    // 24 hours
}
```

## Database Schema Notes

### Fields Used (No Schema Changes Required)

- `users.phone_verified_at` - Phone verification check
- `complaint_attachments` - Live capture check (currently checks for any attachment)
- `complaints.category` - Duplicate detection
- `complaints.latitude` / `complaints.longitude` - Duplicate detection
- `complaints.created_at` - Duplicate detection time window
- `complaints.supporter_count` - Incremented for merged duplicates
- `complaint_supporters` - Stores duplicate relationships

### Future Schema Enhancements

1. **Add `live_capture` field** to `complaint_attachments`:
   ```sql
   ALTER TABLE complaint_attachments 
   ADD COLUMN live_capture BOOLEAN NOT NULL DEFAULT FALSE;
   ```

2. **Add spatial index** for duplicate detection:
   ```sql
   CREATE SPATIAL INDEX idx_complaints_location 
   ON complaints(latitude, longitude);
   ```

3. **Add index** for duplicate queries:
   ```sql
   CREATE INDEX idx_complaints_category_time 
   ON complaints(category, created_at);
   ```

## Testing

See `VERIFICATION_RULES.md` for detailed testing examples including:
- Successful verification
- Duplicate detection and merging
- Failure scenarios for each rule

## Key Design Decisions

1. **No Silent Rejections**: All failures return reason codes, stored in audit log
2. **Duplicate Merging**: Duplicates increment supporter count instead of rejecting
3. **Immutable Audit Trail**: All decisions logged in `audit_log`
4. **Status History**: Status changes go through `complaint_status_history`
5. **Separate Service**: Verification is a separate service/module (not mixed with complaint service)
6. **Rule-Based**: No AI logic - all decisions are rule-based
7. **Configurable**: All thresholds and rules are configurable

## Integration Points

1. **Complaint Creation**: After complaint is created with status "submitted", call verification
2. **Status Updates**: "verified" status can transition to "under_review" or "in_progress"
3. **Audit Logging**: All verification decisions logged automatically
4. **Status History**: Status change to "verified" creates history entry

## Performance Considerations

1. **Duplicate Detection**: Uses Haversine formula (O(n) where n = complaints in time window)
2. **Indexing**: Recommend indexes on `(category, created_at)` and spatial index on coordinates
3. **Audit Logging**: Non-blocking (failures don't stop verification)
4. **Batch Processing**: Can be extended to verify multiple complaints in parallel

## Future Enhancements

1. Add `live_capture` field to schema
2. Use PostGIS/MySQL spatial functions for better duplicate detection
3. Add ML-based duplicate detection (separate from rule-based)
4. Image analysis to verify live capture authenticity
5. Reputation-based verification weighting
6. Batch verification API
