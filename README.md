# Public Accountability System - Complaint Lifecycle API

A Go backend implementation for managing complaint lifecycle in a public accountability system.

## Features

- **Create Complaint**: File new complaints with attachments
- **View Complaint**: Retrieve complaint details (citizen view)
- **Status Timeline**: View complete status change history
- **Update Status**: Internal endpoint for status updates (officer/admin)

## Architecture

```
├── models/          # Data models (entities, DTOs)
├── repository/      # Database access layer
├── service/         # Business logic layer
├── handler/         # HTTP request handlers
├── routes/          # Route configuration
├── config/          # Configuration management
└── main.go          # Application entry point
```

## Database Schema

See `database_schema.md` for complete database schema documentation.

## API Endpoints

### 1. Create Complaint
**POST** `/api/v1/complaints`

Creates a new complaint with proper lifecycle initialization.

**Request Headers:**
```
X-User-ID: <user_id>
Content-Type: application/json
```

**Request Body:**
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
  "attachment_urls": [
    "https://example.com/files/photo1.jpg"
  ]
}
```

**Response:**
```json
{
  "complaint_id": 1,
  "complaint_number": "COMP-20260212-abc12345",
  "status": "submitted",
  "message": "Complaint created successfully"
}
```

### 2. Get Complaint by ID
**GET** `/api/v1/complaints/{id}`

Retrieves complaint details (citizen view). Only accessible to the owner or if the complaint is public.

**Request Headers:**
```
X-User-ID: <user_id>
```

**Response:**
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

### 3. Get Status Timeline
**GET** `/api/v1/complaints/{id}/timeline`

Retrieves the complete status change timeline for a complaint.

**Request Headers:**
```
X-User-ID: <user_id>
```

**Response:**
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

### 4. Update Complaint Status
**PATCH** `/api/v1/complaints/{id}/status`

Updates complaint status (internal use only - requires officer/admin authentication).

**Request Headers:**
```
X-Actor-Type: officer
X-Officer-ID: <officer_id>
Content-Type: application/json
```

**Request Body:**
```json
{
  "new_status": "in_progress",
  "assigned_department_id": 5,
  "assigned_officer_id": 10,
  "notes": "Work has started on this complaint"
}
```

**Response:**
```json
{
  "complaint_id": 1,
  "complaint_number": "COMP-20260212-abc12345",
  "old_status": "under_review",
  "new_status": "in_progress",
  "message": "Status updated successfully"
}
```

## Lifecycle Rules

### Status Transitions

The system enforces valid status transitions:

- `draft` → `submitted`, `draft`
- `submitted` → `under_review`, `rejected`, `draft`
- `under_review` → `in_progress`, `rejected`, `escalated`
- `in_progress` → `resolved`, `rejected`, `escalated`
- `resolved` → `closed`
- `rejected` → `closed`, `under_review` (can be reopened)
- `escalated` → `under_review`, `in_progress`
- `closed` → (terminal state)

### Audit Requirements

1. **Every status change** MUST create an entry in `complaint_status_history`
2. **Every action** MUST create an entry in `audit_log`
3. Both tables are **append-only** (immutable)

### Timestamps

- `resolved_at` is automatically set when status becomes `resolved`
- `closed_at` is automatically set when status becomes `closed`

## Setup

### Prerequisites

- Go 1.21 or higher
- MySQL 5.7+ or MariaDB 10.3+
- Database schema created (see `database_schema.md`)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd finalneta
```

2. Install dependencies:
```bash
go mod download
```

3. Configure environment variables:
```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your_password
export DB_NAME=finalneta
export SERVER_PORT=8080
```

4. Run the application:
```bash
go run main.go
```

## Authentication

**Note**: This implementation assumes authentication middleware exists. In production:

1. Implement JWT or session-based authentication
2. Set `user_id` in request context from authenticated token
3. For internal endpoints, verify officer/admin permissions
4. Update `getUserIDFromContext()` and `getActorFromContext()` in `handler/complaint_handler.go`

Currently, the system expects:
- `X-User-ID` header for user identification
- `X-Actor-Type` header for status updates (values: `user`, `officer`, `system`, `admin`)
- `X-Officer-ID` header when actor type is `officer`

## Error Responses

All errors follow this format:

```json
{
  "error": "Error type",
  "message": "Detailed error message",
  "code": 400
}
```

Common HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request (validation errors)
- `401` - Unauthorized
- `404` - Not Found
- `500` - Internal Server Error

## Testing

Example curl commands:

```bash
# Create complaint
curl -X POST http://localhost:8080/api/v1/complaints \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 1" \
  -d '{
    "title": "Test Complaint",
    "description": "Test description",
    "location_id": 123,
    "public_consent_given": true
  }'

# Get complaint
curl http://localhost:8080/api/v1/complaints/1 \
  -H "X-User-ID: 1"

# Get timeline
curl http://localhost:8080/api/v1/complaints/1/timeline \
  -H "X-User-ID: 1"

# Update status
curl -X PATCH http://localhost:8080/api/v1/complaints/1/status \
  -H "Content-Type: application/json" \
  -H "X-Actor-Type: officer" \
  -H "X-Officer-ID: 10" \
  -d '{
    "new_status": "in_progress",
    "notes": "Work started"
  }'
```

## Notes

- **No verification logic**: Phone verification is assumed to be handled elsewhere
- **No escalation logic**: Escalation is handled separately
- **No notifications**: Email/SMS notifications are handled by other services
- **No AI logic**: All decisions are rule-based
- **Media files**: Assumes files are pre-uploaded and URLs are provided

## License

[Your License Here]
