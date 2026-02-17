# Schema–Code Safety

## Why This Exists

Escalation and status-history flows have failed in production when the database schema lagged behind the code: missing columns (`actor_type`, `actor_id`, `reason`), or timestamp/timezone mismatches. The application assumed columns existed and wrote to them, causing runtime errors or incorrect behaviour.

## What We Do to Prevent It

1. **Idempotent migrations**  
   `migrations/0001_complaint_status_history_audit_columns.sql` adds `actor_type`, `actor_id`, and `reason` to `complaint_status_history` only if each column is missing. Safe to run multiple times; no duplicate-column errors.

2. **UTC timestamps**  
   - DSN uses `loc=UTC` so the MySQL driver parses and interprets timestamps in UTC.  
   - Go code uses `time.Now().UTC()` for any time used in DB writes or compared with DB data (e.g. escalation SLA, status history, complaint numbers).  
   This keeps server and DB in one timezone and avoids escalation/SLA bugs from local-time vs UTC mismatch.

3. **Startup validation**  
   Before the server serves traffic, `schema.ValidateRequiredColumns` checks that required columns exist (e.g. `complaint_status_history.actor_type`, `actor_id`, `reason`). If any are missing, the process logs a **fatal error** listing them and exits. So we fail fast at startup instead of failing during an escalation or status change.

## Result

- **Missing columns** → Clear fatal at startup: “Missing required columns (run migrations to fix): …”  
- **Timezone confusion** → Avoided by using UTC in Go and in the DB connection.  
- **Backward compatible** → Migrations are additive and idempotent; validation only checks presence of columns, not their definition.

No architecture or business logic changes; only safety around schema and timezone assumptions.
