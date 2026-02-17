# Frontend-Backend Integration Summary

## Overview

The frontend has been integrated with backend APIs for complaint submission, fetching complaints list, and viewing complaint details with status timeline.

## Integration Points

### 1. Complaint Submission (ReviewScreen)

**API**: `POST /api/v1/complaints`

**Request Mapping**:
```javascript
Frontend State → API Request
{
  summary → title
  description → description
  category → category
  urgency → priority
  location.latitude → latitude
  location.longitude → longitude
  location.location_id → location_id (defaults to 1 if not provided)
  photo.url → attachment_urls[]
}
```

**Response Handling**:
- Success (201): Navigate to complaint detail page with success message
- Error (400): Show validation error, allow retry
- Error (401): Show auth error
- Network Error: Save to pending queue, show retry option

**Key Features**:
- Photo upload handled before submission (if blob exists)
- Retry logic with exponential backoff
- Offline queue for failed submissions
- Clear error messages

---

### 2. Complaints List (ComplaintsListScreen)

**API**: `GET /api/v1/complaints` (with X-User-ID header)

**Note**: Backend currently doesn't filter by user. This is a placeholder implementation.

**Response Handling**:
- Success: Display list of complaints
- Empty: Show empty state with "File First Complaint" button
- Error: Show error message with retry button
- Network Error: Show network error, allow retry

**Key Features**:
- Loading state while fetching
- Error handling with retry
- Empty state handling
- Status badges with color coding

---

### 3. Complaint Details (ComplaintDetailScreen)

**API**: `GET /api/v1/complaints/{id}`

**Response Handling**:
- Success: Display complaint details
- Error (404): Show "Complaint not found"
- Error (401): Show permission error
- Network Error: Show error with retry button

**Key Features**:
- Success message from navigation state
- Error handling with retry
- Lazy loading of timeline (only when expanded)

---

### 4. Status Timeline (ComplaintDetailScreen)

**API**: `GET /api/v1/complaints/{id}/timeline`

**Response Handling**:
- Success: Display timeline entries
- Error: Show error in timeline section, allow retry
- Network Error: Show network error with retry

**Key Features**:
- Loaded only when user expands timeline
- Separate loading state for timeline
- Error handling doesn't block complaint display

---

## Error Handling Strategy

### Error Types & Messages

| Status | Error Type | User Message | Retryable |
|-------|------------|--------------|-----------|
| 0 | Network Error | "Network error. Please check your connection." | Yes |
| 400 | Validation Error | Error message from API | No |
| 401 | Unauthorized | "Please verify your phone number first." | No |
| 403 | Forbidden | "You do not have permission." | No |
| 404 | Not Found | "Complaint not found." | No |
| 500+ | Server Error | "Server error. Please try again later." | Yes |

### Retry Logic

**Automatic Retries**:
- Network errors: 3 retries with exponential backoff (1s, 2s, 4s)
- Server errors (5xx): 3 retries with exponential backoff
- Timeout: 30 seconds, then retry

**Manual Retries**:
- Retry button shown for retryable errors
- User can manually retry failed requests

---

## Loading States

### ReviewScreen
- `submitting: true` → Button disabled, "Submitting..." text
- `submitting: false` → Button enabled, "Submit Complaint" text

### ComplaintsListScreen
- `loading: true` → "Loading complaints..." message
- `loading: false` → Display list or empty state

### ComplaintDetailScreen
- `loading: true` → "Loading..." message
- `loading: false` → Display complaint details
- `timelineLoading: true` → "Loading timeline..." when expanded

---

## Network Failure Handling

### Offline Detection
- Uses `navigator.onLine` API
- Shows offline banner at top of app
- Listens for online/offline events

### Offline Behavior

**Complaint Submission**:
1. Network error detected
2. Complaint saved to `pending_submissions` in localStorage
3. Error message: "Network error. Your complaint is saved locally and will be submitted when online."
4. Retry button available

**Data Fetching**:
1. Network error detected
2. Show error message
3. Provide retry button
4. Don't crash app

---

## Success Handling

### Complaint Submission Success

**Flow**:
1. API returns `{ complaint_id, complaint_number, status, message }`
2. Clear complaint draft from state and localStorage
3. Navigate to `/complaints/{complaint_id}`
4. Pass success message via navigation state
5. Display success message on detail page

**Implementation**:
```javascript
navigate(`/complaints/${response.complaint_id}`, {
  state: { 
    success: true,
    message: response.message || 'Complaint submitted successfully!'
  }
});
```

---

## API Request/Response Examples

### Create Complaint

**Request**:
```http
POST /api/v1/complaints
Content-Type: application/json
X-User-ID: 1234567890

{
  "title": "Pothole on Main Street",
  "description": "Large pothole causing traffic issues",
  "category": "infrastructure",
  "location_id": 1,
  "latitude": 28.6139,
  "longitude": 77.2090,
  "priority": "high",
  "public_consent_given": true,
  "attachment_urls": []
}
```

**Response**:
```json
{
  "complaint_id": 1,
  "complaint_number": "COMP-20260212-abc12345",
  "status": "submitted",
  "message": "Complaint created successfully"
}
```

### Get Complaint

**Request**:
```http
GET /api/v1/complaints/1
X-User-ID: 1234567890
```

**Response**:
```json
{
  "complaint_id": 1,
  "complaint_number": "COMP-20260212-abc12345",
  "title": "Pothole on Main Street",
  "description": "Large pothole causing traffic issues",
  "category": "infrastructure",
  "current_status": "in_progress",
  "priority": "high",
  "created_at": "2026-02-12T10:00:00Z",
  "attachments": []
}
```

### Get Timeline

**Request**:
```http
GET /api/v1/complaints/1/timeline
X-User-ID: 1234567890
```

**Response**:
```json
{
  "complaint_id": 1,
  "complaint_number": "COMP-20260212-abc12345",
  "timeline": [
    {
      "history_id": 1,
      "old_status": null,
      "new_status": "submitted",
      "changed_by_type": "user",
      "created_at": "2026-02-12T10:00:00Z"
    }
  ]
}
```

---

## Key Implementation Details

### 1. Photo Upload

**Current Implementation**:
- Photo stored as blob URL in state
- On submission, if blob exists, uploads to `/api/v1/upload` endpoint
- Uses uploaded URL in `attachment_urls`
- If upload fails, continues without photo (doesn't fail submission)

**Note**: Backend upload endpoint may not exist yet. Photo upload is optional.

### 2. Location Handling

**Current Implementation**:
- Location captured as `{ latitude, longitude, accuracy }`
- `location_id` defaults to 1 (Shivpuri district) if not provided
- Backend requires `location_id`, so coordinates alone may not work
- **TODO**: Backend should accept coordinates and resolve location_id, or frontend should resolve location_id from coordinates

### 3. User ID Handling

**Current Implementation**:
- User ID stored in localStorage as `user_id` or `user_phone`
- Sent in `X-User-ID` header for all requests
- Backend expects this header for authentication

**Note**: In production, this should come from proper authentication system.

### 4. Retry Strategy

**Implementation**:
- Automatic retries in `requestWithRetry()` function
- Max 3 retries
- Exponential backoff: 1s, 2s, 4s
- Only retries network errors and server errors (5xx)
- Manual retry buttons for user-initiated retries

---

## Testing Checklist

### Complaint Submission
- [ ] Submit with all fields → Success
- [ ] Submit without location → Error (or default location_id)
- [ ] Submit with photo → Photo uploaded first
- [ ] Network error → Saved to pending queue
- [ ] Validation error → Error message shown
- [ ] Success → Navigate to detail page

### Complaints List
- [ ] Load list → Display complaints
- [ ] Empty list → Show empty state
- [ ] Network error → Show error + retry
- [ ] Retry button → Reloads list

### Complaint Details
- [ ] Load details → Display complaint
- [ ] Expand timeline → Load timeline
- [ ] Network error → Show error + retry
- [ ] 404 error → Show "not found"
- [ ] Success message → Display from navigation state

### Error Handling
- [ ] Network error → Retry automatically (3 times)
- [ ] Server error → Retry automatically (3 times)
- [ ] Validation error → Show error, no retry
- [ ] Auth error → Show auth message, no retry

---

## Known Limitations

1. **User Complaints List**: Backend doesn't have dedicated endpoint. Currently returns empty array or all complaints.

2. **Location ID**: Frontend only captures coordinates, but backend requires `location_id`. Using default value (1) for pilot.

3. **Photo Upload**: Upload endpoint may not exist. Photo upload is optional and won't fail submission.

4. **Authentication**: Using localStorage for user ID. Should use proper auth system in production.

5. **Offline Queue**: Pending submissions saved but not auto-synced yet. Manual retry required.

---

## Next Steps

1. **Backend**: Add GET `/api/v1/complaints` endpoint that filters by user_id from header
2. **Backend**: Add POST `/api/v1/upload` endpoint for photo uploads
3. **Backend**: Accept coordinates and resolve location_id, or provide location lookup API
4. **Frontend**: Implement offline queue auto-sync when online
5. **Frontend**: Add proper authentication system
6. **Frontend**: Add location lookup/resolution from coordinates
