# Test Escalation Override (SAFE TEST-ONLY)

Optional config to make escalation trigger after a short delay (e.g. ~2 minutes) instead of SLA hours. **For testing only.** Default is disabled; normal SLA logic is unchanged when the override is not set.

## Purpose

- Test escalation and SLA flow without waiting 72/120 hours.
- Safe: no database or frontend changes; only the time check is overridden when explicitly enabled.
- Easy to disable: set to `0` or unset the env var.

## How to enable

Set the environment variable:

- **`TEST_ESCALATION_OVERRIDE_MINUTES`** – integer, number of minutes to use instead of `sla_hours`.

If **`TEST_ESCALATION_OVERRIDE_MINUTES > 0`**: the escalation worker uses this value (in minutes) as the effective SLA.  
If **`TEST_ESCALATION_OVERRIDE_MINUTES`** is `0` or unset: normal `sla_hours` from escalation rules are used (no override).

## Example: ~2 minutes

```bash
# Linux/macOS
export TEST_ESCALATION_OVERRIDE_MINUTES=2

# Windows (PowerShell)
$env:TEST_ESCALATION_OVERRIDE_MINUTES=2
```

Or in a `.env` file (if you use one):

```env
TEST_ESCALATION_OVERRIDE_MINUTES=2
```

With this, escalation is evaluated against 2 minutes instead of the rule’s `sla_hours` (e.g. 72 or 120 hours).

## Behavior

- **Effective SLA**
  - If `TEST_ESCALATION_OVERRIDE_MINUTES > 0`:  
    `effective_sla = TEST_ESCALATION_OVERRIDE_MINUTES` (minutes).
  - Else:  
    `effective_sla = sla_hours * 60` (minutes).
- The worker compares **time since last status change** (in minutes) to **effective_sla** (in minutes). Escalation is allowed when elapsed time ≥ effective_sla.

## Safety

- **Default**: Override is **disabled** (default `0` in config).
- **No DB or frontend changes**: Only the escalation time check uses the override.
- **No change when disabled**: If the override is not set or is `0`, behavior is exactly the same as before (normal `sla_hours`).

## Logging

On startup the backend logs one of:

- `Test escalation override ENABLED: X minutes`
- `Test escalation override DISABLED`

So you can confirm from logs whether the override is active.

## Config location

- **Config**: `config/config.go` – `PilotConfig.TestEscalationOverrideMinutes`, loaded from `TEST_ESCALATION_OVERRIDE_MINUTES` (default `0`).
- **Usage**: `service/escalation_service.go` – `evaluateEscalationConditions()`: effective SLA and comparison are done in minutes; override is applied only when `TEST_ESCALATION_OVERRIDE_MINUTES > 0`.

## Disable for real pilot

Unset or set to `0`:

```bash
unset TEST_ESCALATION_OVERRIDE_MINUTES
# or
export TEST_ESCALATION_OVERRIDE_MINUTES=0
```

Then restart the backend. Escalation will use normal `sla_hours` again.
