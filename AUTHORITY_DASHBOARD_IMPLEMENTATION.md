# Authority Dashboard Backend Implementation

## Overview
Minimal backend implementation for Department/Authority Dashboard. Allows authorities to view assigned complaints, update status, and add internal notes.

## Database Schema

### New Tables

#### `authority_credentials`
- Stores authentication credentials for authorities
- Supports email+password OR static token (pilot-safe)
- Links to `officers` table via `officer_id`

#### `authority_notes`
- Internal notes added by authorities
- NOT visible to citizens (`is_visible_to_citizen = FALSE`)
- Links to `complaints` and `officers` tables

**SQL File**: `database_authority.sql`

## API Endpoints

### Authentication

#### `POST /api/v1/authority/login`
- **Purpose**: Authority login (pilot: email+password OR static token)
- **Request Body**:
  ```json
  {
    "email": "officer@example.com",
    "password": "password123"
  }
  ```
  OR
  ```json
  {
    "static_token": "pilot-token-123"
  }
  ```
- **Response**:
  ```json
  {
    "success": true,
    "token": "jwt-token-here",
    "officer_id": 1,
    "message": "Login successful"
  }
  ```

### Protected Endpoints (Require `Authorization: Bearer <token>`)

#### `GET /api/v1/authority/complaints`
- **Purpose**: Get complaints assigned to logged-in authority
- **Response**: Array of `ComplaintSummary` objects
- **Filter**: Only returns complaints where `assigned_officer_id` matches logged-in officer
- **Sort**: `created_at DESC`
- **Excludes**: `closed` and `rejected` statuses

#### `POST /api/v1/authority/complaints/{id}/status`
- **Purpose**: Update complaint status with reason
- **Request Body**:
  ```json
  {
    "new_status": "in_progress",
    "reason": "Work has started on this complaint"
  }
  ```
- **Valid Status Transitions**:
  - `under_review` → `in_progress`
  - `in_progress` → `resolved`
  - `resolved` → `closed`
- **Validation**:
  - Complaint must be assigned to logged-in officer
  - Status transition must be valid
  - Reason is required
- **Side Effects**:
  - Creates entry in `complaint_status_history`
  - Sets `resolved_at` when status becomes `resolved`
  - Sets `closed_at` when status becomes `closed`
  - Creates audit log entry

#### `POST /api/v1/authority/complaints/{id}/note`
- **Purpose**: Add internal note (not visible to citizen)
- **Request Body**:
  ```json
  {
    "note_text": "Internal note about this complaint"
  }
  ```
- **Validation**:
  - Complaint must be assigned to logged-in officer
  - Note text is required
- **Side Effects**:
  - Creates entry in `authority_notes`
  - Creates audit log entry

## Authentication Flow

### JWT Token Structure
- **Claims**:
  - `officer_id`: Officer ID
  - `actor_type`: "authority" (to distinguish from citizen tokens)
  - `exp`: Expiration timestamp
  - `iat`: Issued at timestamp
- **Expiration**: 7 days (configurable)

### Middleware
- `AuthorityAuthMiddleware`: Validates JWT token, extracts `officer_id`, sets in request context
- Separates authority authentication from citizen authentication

## Status Transition Rules

### Authority-Allowed Transitions
1. `under_review` → `in_progress`
2. `in_progress` → `resolved`
3. `resolved` → `closed`

### Enforcement
- Status transitions are validated in `AuthorityService.UpdateComplaintStatus`
- Invalid transitions return `400 Bad Request` with error message
- Cannot skip statuses (e.g., cannot go from `under_review` directly to `resolved`)

## Files Created/Modified

### New Files
1. `database_authority.sql` - Database schema for authority tables
2. `handler/authority_handler.go` - HTTP handlers for authority endpoints
3. `handler/authority_auth_handler.go` - Authority login handler
4. `service/authority_service.go` - Business logic for authority operations
5. `repository/authority_repository.go` - Database operations for authority
6. `middleware/authority_auth.go` - JWT authentication middleware for authorities

### Modified Files
1. `models/dtos.go` - Added DTOs:
   - `AuthorityUpdateStatusRequest`
   - `AuthorityAddNoteRequest`
   - `AuthorityNoteResponse`
   - `AuthorityNote`
2. `utils/jwt.go` - Added `GenerateAuthorityJWT()` function
3. `repository/complaint_repository.go` - Added `UpdateComplaintStatusWithTimestamps()` method
4. `routes/routes.go` - Added authority routes
5. `main.go` - Initialize authority repository and pass to routes

## Security Notes

### Pilot Implementation
- **Static Token**: Supported for pilot convenience
- **Password Storage**: Simple comparison (NOT production-ready)
- **JWT Secret**: Uses same secret as citizen auth (can be separated)

### Production Recommendations
1. Use `bcrypt` or `argon2` for password hashing
2. Remove static token support
3. Use separate JWT secret for authorities
4. Implement rate limiting on login endpoint
5. Add account lockout after failed attempts
6. Use HTTPS only

## Usage Example

### 1. Login
```bash
curl -X POST http://localhost:8080/api/v1/authority/login \
  -H "Content-Type: application/json" \
  -d '{"email": "officer@example.com", "password": "password123"}'
```

### 2. Get Assigned Complaints
```bash
curl -X GET http://localhost:8080/api/v1/authority/complaints \
  -H "Authorization: Bearer <token>"
```

### 3. Update Status
```bash
curl -X POST http://localhost:8080/api/v1/authority/complaints/123/status \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"new_status": "in_progress", "reason": "Work started"}'
```

### 4. Add Note
```bash
curl -X POST http://localhost:8080/api/v1/authority/complaints/123/note \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"note_text": "Internal note"}'
```

## Database Setup

Run the SQL file to create tables:
```sql
SOURCE database_authority.sql;
```

Or manually execute the SQL statements in `database_authority.sql`.

## Next Steps

1. **Seed Authority Credentials**: Create initial authority accounts
   ```sql
   INSERT INTO authority_credentials (officer_id, email, password_hash, static_token)
   VALUES (1, 'officer@example.com', 'password123', 'pilot-token-123');
   ```

2. **Test Authentication**: Verify login endpoint works

3. **Test Protected Endpoints**: Verify complaints list, status update, and notes work

4. **Frontend Integration**: Build authority dashboard UI (separate task)
