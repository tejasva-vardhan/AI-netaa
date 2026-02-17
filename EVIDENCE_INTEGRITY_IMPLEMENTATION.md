# Evidence Integrity Verification Implementation

## Overview

Implements a server-side, write-once evidence hash for complaint media (photos). The hash is an **integrity signal** to detect tampering after capture; it is **not** authenticity proof (see Trust model below).

## Requirements

1. **Hash input**: RAW image bytes + latitude + longitude + server-generated `captured_at`. No hex string, no URL-derived content.
2. **Timestamp**: Server-side only. Do not accept client-provided timestamps. Use server-generated `captured_at` at upload time.
3. **No fetch-from-URL**: Hash must be computed at upload time, before persistence. Do not fetch image from storage URLs for hashing (URL content can change; compression may differ).
4. **Write-once**: `complaint_evidence` is immutable after insert. No updates to `evidence_hash`, `latitude`, `longitude`, or `captured_at`. Enforced at service layer (no update method).

## Trust Model (Legal / Compliance)

- **Evidence hash = integrity signal**: It allows detection of changes to the image or to the recorded lat/lng/timestamp after the record was created. It does **not** prove who took the photo, when the device actually captured it, or that the location is truthful.
- **Do not overclaim**: In legal or compliance contexts, describe the hash as an integrity mechanism only, not as proof of authenticity, origin, or chain of custody beyond what the system records.

## Database Schema

### Table: `complaint_evidence` (write-once)

```sql
CREATE TABLE IF NOT EXISTS complaint_evidence (
    evidence_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    attachment_id BIGINT NOT NULL COMMENT 'Related attachment (photo)',
    complaint_id BIGINT NOT NULL COMMENT 'Related complaint',
    evidence_hash VARCHAR(64) NOT NULL COMMENT 'SHA256 of raw image_bytes + lat + lng + server captured_at',
    captured_at TIMESTAMP NOT NULL COMMENT 'Server-side timestamp at upload (do not use client time)',
    latitude DECIMAL(10, 8) NULL COMMENT 'GPS latitude at capture time',
    longitude DECIMAL(11, 8) NULL COMMENT 'GPS longitude at capture time',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Record creation',
    ...
);
```

**SQL File**: `database_evidence_integrity.sql`

- No UPDATE statements are exposed; application uses only INSERT for this table.

## Hash Generation

### Input (raw bytes only)

- **image_bytes**: Raw image bytes (as received at upload).
- **latitude**: float64, little-endian (8 bytes).
- **longitude**: float64, little-endian (8 bytes).
- **captured_at**: Server-generated timestamp at upload; encoded as Unix nanoseconds (int64, little-endian).

Concatenation order: `image_bytes || latitude || longitude || captured_at_unix_nano`.

### Algorithm

1. Start with raw `image_bytes`.
2. Append latitude as 8 bytes (float64, little-endian).
3. Append longitude as 8 bytes (float64, little-endian).
4. Append `capturedAt.UnixNano()` as 8 bytes (int64, little-endian).
5. SHA-256 the resulting byte slice.
6. Store hex-encoded hash (64 characters).

### Implementation

- **File**: `utils/evidence_hash.go`
- **Function**: `GenerateEvidenceHash(imageBytes, latitude, longitude, capturedAt time.Time) string`
- **Timestamp**: Caller must pass server time (e.g. `time.Now()`). Client timestamps must not be used.

There is no verification function in the pilot phase; integrity is ensured by immutability and audit trail.

## Service Layer

- **File**: `service/evidence_service.go`
- **CreateEvidenceRecord**: Creates the evidence row. Uses server-generated `capturedAt`. No update method is provided; write-once is enforced at the service layer.

## Integration (Option A only)

- Hash must be generated **at upload time**, when raw image bytes are available, **before** persisting the file or attachment.
- Do **not** derive the hash from URLs or by re-fetching the image from storage.

Example flow:

1. Upload endpoint receives multipart form with image file + optional lat/lng.
2. Read image into `[]byte`.
3. `capturedAt := time.Now()` (server).
4. `hash := utils.GenerateEvidenceHash(imageBytes, lat, lng, capturedAt)`.
5. Persist file/attachment as usual.
6. Insert into `complaint_evidence`: `evidence_hash`, `captured_at`, `latitude`, `longitude`, plus `attachment_id` and `complaint_id` when available.

## Files Touched

| File | Purpose |
|------|---------|
| `database_evidence_integrity.sql` | Write-once evidence table |
| `utils/evidence_hash.go` | Hash from raw bytes + lat + lng + server time; no verification |
| `service/evidence_service.go` | Create evidence only; write-once documented |
| `repository/evidence_repository.go` | Create only; no update |

## Security and Compliance Notes

- **Server-side only**: Hash and `captured_at` are generated on the server.
- **No client timestamp**: `captured_at` is never taken from the client.
- **Write-once**: No updates to `evidence_hash`, `latitude`, `longitude`, or `captured_at` after insert.
- **Integrity, not authenticity**: Hash is an integrity signal; do not use it as sole proof of authenticity in legal or compliance claims.
