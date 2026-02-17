# Complaint Verification Engine

## Overview

The verification engine is a rule-based system that automatically verifies complaints before they can proceed to the review stage. Verification ensures data quality, prevents duplicates, and maintains system integrity.

## Verification Flow

```
Complaint Created (status: "submitted")
    ↓
Verification Engine Runs
    ↓
All Rules Pass? → Yes → Status: "verified"
    ↓ No
Reason Code Stored in Audit Log
```

## Verification Rules

### Rule 1: Live Capture Attachment

**Requirement**: Complaint must have at least one attachment marked as `live_capture = true`

**Purpose**: Ensures complaints have real-time evidence (photos/videos taken at the time of filing)

**Implementation**:
- Checks `complaint_attachments` table for attachments linked to the complaint
- In production, this would check a `live_capture` field or metadata JSON field
- Current implementation assumes at least one attachment means live capture exists

**Failure Reason Code**: `NO_LIVE_CAPTURE`

**Example**:
```json
{
  "verified": false,
  "reason_code": "NO_LIVE_CAPTURE",
  "reason_message": "No attachment with live_capture=true found"
}
```

---

### Rule 2: GPS Accuracy

**Requirement**: GPS accuracy must be within acceptable range (configurable, default: ≤ 100 meters)

**Purpose**: Ensures location data is reliable for duplicate detection and routing

**Implementation**:
- GPS accuracy is passed in the verification request (optional)
- If provided, must be ≤ configured threshold (default: 100 meters)
- If not provided, rule is skipped (assumes accuracy is acceptable)

**Configuration**:
```go
config := &VerificationConfig{
    GPSAccuracyThreshold: 100.0, // meters
}
```

**Failure Reason Code**: `GPS_ACCURACY_EXCEEDED`

**Example**:
```json
{
  "verified": false,
  "reason_code": "GPS_ACCURACY_EXCEEDED",
  "reason_message": "GPS accuracy 150.00 meters exceeds threshold of 100.00 meters"
}
```

---

### Rule 3: Phone Verification

**Requirement**: User phone number must be verified

**Purpose**: Ensures complaints come from verified users, reducing spam and fake complaints

**Implementation**:
- Checks `users.phone_verified_at` field
- Must be NOT NULL (phone has been verified)

**Failure Reason Code**: `PHONE_NOT_VERIFIED`

**Example**:
```json
{
  "verified": false,
  "reason_code": "PHONE_NOT_VERIFIED",
  "reason_message": "User phone number is not verified"
}
```

---

### Rule 4: Duplicate Detection

**Requirement**: No duplicate complaints found within:
- Same category (if category is provided)
- Same location (within configurable radius, default: 50 meters)
- Within configurable time window (default: 24 hours)

**Purpose**: Prevents duplicate complaints and consolidates similar issues

**Duplicate Detection Logic**:

1. **Category Match**:
   - If complaint has a category, only check complaints with same category
   - If complaint has no category, only check complaints with no category

2. **Location Match**:
   - Calculate distance using Haversine formula
   - Only consider complaints within radius (default: 50 meters)

3. **Time Window**:
   - Only check complaints created within time window (default: 24 hours)
   - Excludes complaints with status 'rejected' or 'closed'

4. **Merging Process**:
   - If duplicate found, find the oldest complaint (original)
   - Increment `supporter_count` on original complaint
   - Add current user as supporter with `is_duplicate = true`
   - Store reference to original complaint in `duplicate_notes`
   - **Do NOT create new complaint**
   - Return result with `duplicate_complaint_id` and updated `supporter_count`

**Configuration**:
```go
config := &VerificationConfig{
    DuplicateDetectionRadius:     50.0,        // meters
    DuplicateDetectionTimeWindow:  24 * time.Hour, // 24 hours
}
```

**Failure Reason Code**: `DUPLICATE_FOUND`

**Example**:
```json
{
  "verified": false,
  "reason_code": "DUPLICATE_FOUND",
  "reason_message": "Duplicate complaint found. Merged with complaint #123",
  "duplicate_complaint_id": 123,
  "supporter_count": 5
}
```

**Duplicate Merging Details**:

When a duplicate is detected:
1. Original complaint's `supporter_count` is incremented
2. New entry in `complaint_supporters` table:
   - `complaint_id`: Original complaint ID
   - `user_id`: User who filed the duplicate
   - `is_duplicate`: `true`
   - `duplicate_notes`: "Merged from complaint #<new_complaint_id>"
3. Audit log entry created for both complaints
4. New complaint remains in "submitted" status (not verified)

---

## Verification Result Codes

| Code | Description | Action |
|------|-------------|--------|
| `VERIFIED` | All rules passed | Status updated to "verified" |
| `NO_LIVE_CAPTURE` | No live capture attachment found | Complaint remains "submitted" |
| `GPS_ACCURACY_EXCEEDED` | GPS accuracy exceeds threshold | Complaint remains "submitted" |
| `PHONE_NOT_VERIFIED` | User phone not verified | Complaint remains "submitted" |
| `DUPLICATE_FOUND` | Duplicate complaint found | Merged with original, supporter count incremented |

---

## API Endpoint

### POST `/api/v1/complaints/{id}/verify`

Verifies a complaint according to all verification rules.

**Request**:
```http
POST /api/v1/complaints/123/verify
Content-Type: application/json

{
  "gps_accuracy": 45.5  // Optional: GPS accuracy in meters
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

**Response (Failure - No Live Capture)**:
```json
{
  "complaint_id": 123,
  "verified": false,
  "reason_code": "NO_LIVE_CAPTURE",
  "reason_message": "No attachment with live_capture=true found"
}
```

**Response (Failure - Duplicate Found)**:
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

---

## Audit Logging

**Every verification decision is logged in `audit_log`**:

```json
{
  "entity_type": "complaint",
  "entity_id": 123,
  "action": "verification",
  "action_by_type": "system",
  "metadata": {
    "verification": {
      "verified": true,
      "reason_code": "VERIFIED",
      "reason_message": "Complaint verified successfully",
      "verification_at": "2026-02-12T10:00:00Z"
    },
    "rules_passed": [
      "live_capture_attachment",
      "gps_accuracy",
      "phone_verified",
      "no_duplicates"
    ],
    "gps_accuracy": 45.5
  }
}
```

---

## Status Transitions

### After Verification

- **Success**: `submitted` → `verified`
- **Failure**: `submitted` → `submitted` (no change, reason stored in audit log)

### From Verified Status

- `verified` → `under_review` (manual review)
- `verified` → `in_progress` (work started)
- `verified` → `rejected` (if invalid)

---

## Configuration

Default configuration can be customized:

```go
config := &models.VerificationConfig{
    GPSAccuracyThreshold:         100.0,              // meters
    DuplicateDetectionRadius:     50.0,              // meters
    DuplicateDetectionTimeWindow: 24 * time.Hour,    // 24 hours
}

verificationService := service.NewVerificationService(
    complaintRepo,
    verificationRepo,
    config,
)
```

---

## Implementation Notes

### Database Schema Compatibility

Since the schema cannot be changed, the implementation works with existing fields:

1. **Live Capture**: Currently checks for any attachment. In production, add `live_capture` boolean field to `complaint_attachments` or use metadata JSON.

2. **GPS Accuracy**: Passed in verification request. Could be stored in complaint metadata JSON if needed.

3. **Phone Verification**: Uses existing `users.phone_verified_at` field.

4. **Duplicate Detection**: Uses existing `complaints` table fields (category, latitude, longitude, created_at).

### Distance Calculation

Uses Haversine formula to calculate distance between coordinates:

```
distance = 2 * R * arcsin(√(sin²(Δlat/2) + cos(lat1) * cos(lat2) * sin²(Δlon/2)))
```

Where R = Earth radius (6,371,000 meters)

### Performance Considerations

1. **Duplicate Detection**: 
   - Index on `(category, created_at)` recommended
   - Index on `(latitude, longitude)` for spatial queries
   - Consider using PostGIS or MySQL spatial indexes for production

2. **Audit Logging**:
   - Audit log writes are non-blocking (failures don't stop verification)
   - Consider async logging for high-volume systems

3. **Status History**:
   - Every status change creates immutable history entry
   - Index on `(complaint_id, created_at DESC)` for timeline queries

---

## Testing Examples

### Test Case 1: Successful Verification

```bash
curl -X POST http://localhost:8080/api/v1/complaints/123/verify \
  -H "Content-Type: application/json" \
  -d '{"gps_accuracy": 45.5}'
```

**Expected**: Status changes to "verified"

### Test Case 2: Duplicate Detection

```bash
# Create first complaint
curl -X POST http://localhost:8080/api/v1/complaints \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 1" \
  -d '{
    "title": "Pothole on Main St",
    "description": "Large pothole",
    "category": "infrastructure",
    "location_id": 100,
    "latitude": 28.6139,
    "longitude": 77.2090
  }'

# Verify first complaint (should succeed)
curl -X POST http://localhost:8080/api/v1/complaints/1/verify

# Create duplicate complaint (same location, within 24 hours)
curl -X POST http://localhost:8080/api/v1/complaints \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 2" \
  -d '{
    "title": "Pothole on Main St",
    "description": "Same pothole",
    "category": "infrastructure",
    "location_id": 100,
    "latitude": 28.6140,
    "longitude": 77.2091
  }'

# Verify duplicate complaint (should merge)
curl -X POST http://localhost:8080/api/v1/complaints/2/verify
```

**Expected**: Complaint #2 merged with #1, supporter_count incremented

---

## Error Handling

- Verification failures are **not errors** - they return 200 OK with `verified: false`
- Reason codes explain why verification failed
- All decisions are logged in audit_log for transparency
- Duplicate detection merges complaints instead of rejecting

---

## Future Enhancements

1. **Machine Learning**: Add ML-based duplicate detection (separate from rule-based)
2. **Image Analysis**: Verify live capture images are not stock photos
3. **Location Validation**: Cross-reference with official location databases
4. **Reputation System**: Weight verification based on user reputation
5. **Batch Verification**: Verify multiple complaints in parallel
