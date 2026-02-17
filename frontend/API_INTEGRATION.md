# API Integration Guide

## Overview

This document describes how the frontend integrates with backend APIs, including request/response mapping, error handling, and retry strategies.

## API Endpoints

### 1. Create Complaint

**Endpoint**: `POST /api/v1/complaints`

**Request Headers**:
```
Content-Type: application/json
X-User-ID: <user_id>
```

**Request Body**:
```json
{
  "title": "Pothole on Main Street",
  "description": "Large pothole causing traffic issues",
  "category": "infrastructure",
  "location_id": 123,
  "latitude": 28.6139,
  "longitude": 77.2090,
  "priority": "high",
  "public_consent_given": true,
  "attachment_urls": ["https://example.com/files/photo1.jpg"]
}
```

**Response** (Success - 201):
```json
{
  "complaint_id": 1,
  "complaint_number": "COMP-20260212-abc12345",
  "status": "submitted",
  "message": "Complaint created successfully"
}
```

**Response** (Error - 400):
```json
{
  "error": "Validation error",
  "message": "Title is required",
  "code": 400
}
```

**Frontend Mapping**:
```javascript
// Input: complaintData from state
const requestBody = {
  title: complaintData.summary,
  description: complaintData.description,
  category: complaintData.category || null,
  location_id: complaintData.location?.location_id || null,
  latitude: complaintData.location?.latitude || null,
  longitude: complaintData.location?.longitude || null,
  priority: complaintData.urgency || 'medium',
  public_consent_given: true,
  attachment_urls: attachmentUrls
};

// Output: Navigate to /complaints/{complaint_id}
```

---

### 2. Get Complaint by ID

**Endpoint**: `GET /api/v1/complaints/{id}`

**Request Headers**:
```
X-User-ID: <user_id>
```

**Response** (Success - 200):
```json
{
  "complaint_id": 1,
  "complaint_number": "COMP-20260212-abc12345",
  "title": "Pothole on Main Street",
  "description": "Large pothole causing traffic issues",
  "category": "infrastructure",
  "location_id": 123,
  "latitude": 28.6139,
  "longitude": 77.2090,
  "assigned_department_id": 5,
  "assigned_officer_id": 10,
  "current_status": "in_progress",
  "priority": "high",
  "is_public": true,
  "supporter_count": 5,
  "created_at": "2026-02-12T10:00:00Z",
  "attachments": [
    {
      "attachment_id": 1,
      "file_name": "photo1.jpg",
      "file_path": "https://example.com/files/photo1.jpg",
      "file_type": "image/jpeg",
      "file_size": 102400,
      "is_public": true
    }
  ]
}
```

**Response** (Error - 404):
```json
{
  "error": "Not found",
  "message": "complaint not found",
  "code": 404
}
```

**Frontend Mapping**:
- Display complaint details
- Show attachments if available
- Format dates for display

---

### 3. Get Complaint Timeline

**Endpoint**: `GET /api/v1/complaints/{id}/timeline`

**Request Headers**:
```
X-User-ID: <user_id>
```

**Response** (Success - 200):
```json
{
  "complaint_id": 1,
  "complaint_number": "COMP-20260212-abc12345",
  "timeline": [
    {
      "history_id": 3,
      "old_status": "under_review",
      "new_status": "in_progress",
      "changed_by_type": "officer",
      "changed_by_officer_id": 10,
      "assigned_department_id": 5,
      "assigned_officer_id": 10,
      "notes": "Work started",
      "created_at": "2026-02-12T14:00:00Z"
    },
    {
      "history_id": 2,
      "old_status": "submitted",
      "new_status": "under_review",
      "changed_by_type": "system",
      "assigned_department_id": 5,
      "created_at": "2026-02-12T11:00:00Z"
    },
    {
      "history_id": 1,
      "old_status": null,
      "new_status": "submitted",
      "changed_by_type": "user",
      "changed_by_user_id": 1,
      "notes": "Complaint created",
      "created_at": "2026-02-12T10:00:00Z"
    }
  ]
}
```

**Frontend Mapping**:
- Display timeline in chronological order (newest first)
- Show status transitions
- Display notes if available
- Format timestamps

---

### 4. Get User's Complaints

**Endpoint**: `GET /api/v1/complaints` (with X-User-ID header)

**Note**: Backend currently doesn't have dedicated endpoint. This is a placeholder.

**Expected Response** (Success - 200):
```json
[
  {
    "complaint_id": 1,
    "complaint_number": "COMP-20260212-abc12345",
    "title": "Pothole on Main Street",
    "current_status": "in_progress",
    "priority": "high",
    "created_at": "2026-02-12T10:00:00Z",
    "supporter_count": 5
  }
]
```

**Frontend Mapping**:
- Display list of complaints
- Show status badges
- Format dates
- Handle empty list

---

## Error Handling Strategy

### Error Types

1. **Network Errors** (status: 0)
   - No internet connection
   - Request timeout
   - DNS failure

2. **Client Errors** (4xx)
   - 400: Bad Request (validation errors)
   - 401: Unauthorized (authentication required)
   - 403: Forbidden (no permission)
   - 404: Not Found

3. **Server Errors** (5xx)
   - 500: Internal Server Error
   - 502: Bad Gateway
   - 503: Service Unavailable
   - 504: Gateway Timeout

### Error Handling Flow

```
API Request
  ↓
Network Error?
  ↓ Yes → Retry (exponential backoff, max 3 times)
  ↓ No
HTTP Error?
  ↓ Yes → Check status code
  ↓       400 → Show validation error
  ↓       401 → Show auth error
  ↓       404 → Show not found
  ↓       5xx → Retry (if retryable)
  ↓ No
Success → Process response
```

### Retry Logic

**Retryable Errors**:
- Network errors (status: 0)
- Server errors (500, 502, 503, 504)
- Timeout errors

**Retry Strategy**:
- Max retries: 3
- Exponential backoff: 1s, 2s, 4s
- Only retry on retryable errors

**Implementation**:
```javascript
async function requestWithRetry(endpoint, options = {}, retryCount = 0) {
  try {
    const response = await fetch(url, config);
    // ... handle response
  } catch (error) {
    if (isRetryable(error) && retryCount < maxRetries) {
      const delay = retryDelay * Math.pow(2, retryCount);
      await sleep(delay);
      return requestWithRetry(endpoint, options, retryCount + 1);
    }
    throw error;
  }
}
```

---

## Loading States

### ReviewScreen (Submit)

**States**:
- `submitting: false` → Submit button enabled
- `submitting: true` → Submit button disabled, "Submitting..." text
- `error: null` → No error message
- `error: string` → Error message displayed, retry button shown

**UI Changes**:
```jsx
<button disabled={submitting}>
  {submitting ? 'Submitting...' : 'Submit Complaint'}
</button>
{error && <div className="error">{error}</div>}
{error && <button onClick={handleRetry}>Retry</button>}
```

### ComplaintsListScreen

**States**:
- `loading: true` → "Loading complaints..." message
- `loading: false, error: null` → Display complaints list
- `loading: false, error: string` → Error message, retry button
- `complaints.length === 0` → Empty state

**UI Changes**:
```jsx
{loading && <div>Loading complaints...</div>}
{error && <div className="error">{error}</div>}
{error && <button onClick={handleRetry}>Retry</button>}
{complaints.length === 0 && <EmptyState />}
```

### ComplaintDetailScreen

**States**:
- `loading: true` → "Loading..." message
- `loading: false, error: null` → Display complaint details
- `loading: false, error: string` → Error message, retry button
- `timelineLoading: true` → "Loading timeline..." when timeline expanded

**UI Changes**:
```jsx
{loading && <div>Loading...</div>}
{error && <div className="error">{error}</div>}
{error && <button onClick={handleRetry}>Retry</button>}
{timelineLoading && <div>Loading timeline...</div>}
```

---

## Network Failure Handling

### Offline Detection

```javascript
const [isOnline, setIsOnline] = useState(navigator.onLine);

useEffect(() => {
  window.addEventListener('online', () => setIsOnline(true));
  window.addEventListener('offline', () => setIsOnline(false));
}, []);
```

### Offline Behavior

1. **Complaint Submission**:
   - Save to `pending_submissions` in localStorage
   - Show message: "Network error. Your complaint is saved locally and will be submitted when online."
   - Auto-retry when online (future enhancement)

2. **Data Fetching**:
   - Show error message
   - Provide retry button
   - Don't crash app

3. **Offline Banner**:
   - Display at top of app when offline
   - Message: "No internet connection. Your data will be saved and synced when online."

---

## Success Handling

### Complaint Submission Success

**Flow**:
1. API returns success response
2. Clear complaint draft from state
3. Navigate to complaint detail page
4. Show success message (from navigation state)

**Implementation**:
```javascript
const response = await api.createComplaint(complaintData);
clearComplaintData();
navigate(`/complaints/${response.complaint_id}`, {
  state: { 
    success: true,
    message: response.message || 'Complaint submitted successfully!'
  }
});
```

---

## Example API Request/Response Mapping

### Create Complaint Flow

**Frontend State**:
```javascript
{
  summary: "Pothole on Main Street",
  description: "Large pothole causing traffic issues",
  category: "infrastructure",
  urgency: "high",
  location: {
    latitude: 28.6139,
    longitude: 77.2090,
    location_id: 123
  },
  photo: {
    blob: File,
    url: "blob:..."
  }
}
```

**API Request**:
```javascript
POST /api/v1/complaints
Headers: {
  "Content-Type": "application/json",
  "X-User-ID": "1234567890"
}
Body: {
  "title": "Pothole on Main Street",
  "description": "Large pothole causing traffic issues",
  "category": "infrastructure",
  "location_id": 123,
  "latitude": 28.6139,
  "longitude": 77.2090,
  "priority": "high",
  "public_consent_given": true,
  "attachment_urls": ["https://uploaded-photo-url"]
}
```

**API Response**:
```javascript
{
  complaint_id: 1,
  complaint_number: "COMP-20260212-abc12345",
  status: "submitted",
  message: "Complaint created successfully"
}
```

**Frontend Action**:
- Navigate to `/complaints/1`
- Show success message
- Clear draft data

---

## Error Messages

### User-Friendly Error Messages

| Error Type | Message |
|------------|---------|
| Network Error | "Network error. Please check your connection." |
| Validation Error (400) | "Invalid data. Please check your input." |
| Unauthorized (401) | "Please verify your phone number first." |
| Not Found (404) | "Complaint not found." |
| Server Error (5xx) | "Server error. Please try again later." |
| Timeout | "Request timeout. Please check your connection." |

### Retry Messages

- Network errors: "Network error. [Retry]"
- Server errors: "Server error. [Retry]"
- Timeout: "Request timeout. [Retry]"

---

## Testing Scenarios

### 1. Successful Submission

1. Fill complaint form
2. Click Submit
3. See "Submitting..." state
4. Receive success response
5. Navigate to complaint detail
6. See success message

### 2. Network Error During Submission

1. Fill complaint form
2. Disconnect network
3. Click Submit
4. See network error message
5. Complaint saved to localStorage
6. Retry button available

### 3. Validation Error

1. Fill form with missing required fields
2. Click Submit
3. See validation error message
4. Form remains filled
5. Can correct and retry

### 4. Slow Network

1. Fill complaint form
2. Click Submit (slow network)
3. See "Submitting..." state
4. Request times out after 30s
5. See timeout error
6. Retry button available

### 5. Load Complaints List

1. Navigate to complaints list
2. See loading state
3. Receive complaints data
4. Display list
5. If error, show error + retry button

---

## Future Enhancements

1. **Offline Queue**: Auto-sync pending submissions when online
2. **Optimistic Updates**: Update UI immediately, sync later
3. **Background Sync**: Use Service Worker for background sync
4. **Request Cancellation**: Cancel in-flight requests on navigation
5. **Request Deduplication**: Prevent duplicate requests
6. **Response Caching**: Cache responses for offline viewing
