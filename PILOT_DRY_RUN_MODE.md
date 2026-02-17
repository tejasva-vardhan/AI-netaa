# Pilot Dry-Run / Testing Mode

Internal team testing mode for safe escalation & SLA testing without affecting production data.

## Configuration

### Environment Variables

1. **`PILOT_DRY_RUN`** (boolean, default: `false`)
   - Enable dry-run/testing mode
   - Set to `true` to enable

2. **`PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES`** (integer, default: `0`)
   - Override SLA hours with minutes for faster testing
   - Set to `0` to disable override (use normal hours)
   - Example: `5` = 5 minutes instead of hours

### Example `.env` Configuration

```env
# Enable dry-run mode
PILOT_DRY_RUN=true

# Override SLA to 5 minutes (for testing)
PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES=5
```

## Behavior When Enabled

### 1. SLA Time Override
- When `PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES > 0`:
  - SLA hours are converted to minutes for faster testing
  - Example: 72 hours → 5 minutes (if override = 5)
  - Logs clearly indicate override: `[DRY RUN] SLA override: 72 hours -> 5 minutes (0.08 hours)`

### 2. Email Behavior
- **Emails still go to shadow inbox**: `aineta502@gmail.com`
- No changes to email shadow mode behavior
- All emails are still logged to `email_logs` table

### 3. Escalation Logs
- **Status history**: Reason field prefixed with `[DRY RUN]`
  - Example: `[DRY RUN] Escalated to level 0: SLA breached`
- **Audit log**: Includes `dry_run: true` and `dry_run_sla_override_minutes` in metadata
- **Console logs**: All escalation logs prefixed with `[DRY RUN]`

### 4. Data Safety
- **No data corruption**: All database operations proceed normally
- Escalations are real (status changes, history entries, escalation records)
- Only time calculations and logging are affected
- Easy to disable: Set `PILOT_DRY_RUN=false` or remove env var

## Implementation Details

### Config Structure
**File:** `config/config.go`

```go
type PilotConfig struct {
    DryRun              bool  // PILOT_DRY_RUN
    DryRunSLAOverrideMinutes int // PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES
}
```

### Escalation Service Changes
**File:** `service/escalation_service.go`

1. **SLA Override Logic** (line ~261-275):
   - Checks `dryRun` and `dryRunSLAOverrideMinutes`
   - Converts minutes to hours: `minutes / 60.0`
   - Logs override when applied

2. **Status History Marking** (line ~358):
   - Prefixes reason with `[DRY RUN]` when dry-run enabled

3. **Audit Log Marking** (line ~407-420):
   - Adds `dry_run: true` to audit metadata
   - Includes `dry_run_sla_override_minutes` if override is set

4. **Worker Logging** (line ~55-60):
   - Logs dry-run mode status at worker start
   - Logs SLA override status if configured

### Startup Logging
**File:** `main.go` (line ~26-35)

- Logs dry-run mode status on startup
- Logs SLA override configuration
- Confirms email shadow inbox destination

## Usage Examples

### Enable Dry-Run with 5-Minute SLA Override

```bash
export PILOT_DRY_RUN=true
export PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES=5
```

**Expected behavior:**
- Escalations trigger after 5 minutes instead of 72 hours
- All logs marked with `[DRY RUN]`
- Emails still go to `aineta502@gmail.com`

### Enable Dry-Run without SLA Override

```bash
export PILOT_DRY_RUN=true
# PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES not set (defaults to 0)
```

**Expected behavior:**
- Escalations use normal SLA hours (72, 120)
- All logs marked with `[DRY RUN]`
- Emails still go to `aineta502@gmail.com`

### Disable Dry-Run (Normal Mode)

```bash
unset PILOT_DRY_RUN
# or
export PILOT_DRY_RUN=false
```

**Expected behavior:**
- Normal escalation behavior
- Normal SLA hours
- No `[DRY RUN]` markers

## Testing Scenarios

### 1. Fast Escalation Testing
```env
PILOT_DRY_RUN=true
PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES=2
```
- Create complaint
- Wait 2 minutes
- Escalation should trigger (instead of 72 hours)

### 2. Dry-Run Logging Verification
```env
PILOT_DRY_RUN=true
PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES=5
```
- Check status history: Should see `[DRY RUN]` prefix
- Check audit_log: Should see `dry_run: true` in metadata
- Check console logs: Should see `[DRY RUN]` prefixes

### 3. Email Shadow Mode Verification
```env
PILOT_DRY_RUN=true
```
- Verify emails still go to `aineta502@gmail.com`
- Verify emails are logged in `email_logs` table
- Verify email content is unchanged

## Safety Guarantees

1. **No Data Corruption**
   - All database operations proceed normally
   - Escalations create real records
   - Status changes are real

2. **Easy to Disable**
   - Set `PILOT_DRY_RUN=false` or remove env var
   - No code changes required
   - No database migrations needed

3. **Clear Marking**
   - All dry-run actions clearly marked in logs
   - Status history reasons prefixed with `[DRY RUN]`
   - Audit logs include dry-run metadata

## Notes

- **Internal use only**: This mode is for internal team testing
- **Email shadow mode**: Already sends to pilot inbox, no changes needed
- **Metrics**: Pilot metrics events are still emitted normally
- **Worker**: Escalation worker respects dry-run mode automatically
