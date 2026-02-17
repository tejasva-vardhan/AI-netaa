# Escalation Worker Interval (Config-Driven & Auto-Adaptive)

The escalation worker run interval is **config-driven** and **auto-adaptive**: it stays at 1 hour in production and shortens automatically when the test escalation override is enabled, so you don’t need to restart or manually trigger escalation.

## Config value

**`ESCALATION_WORKER_INTERVAL_SECONDS`** (integer, optional)

- When **not set**: default behavior applies (see below).
- When **set**: used as the worker interval in seconds, subject to pilot rules when test override is on.

## Default (production-safe) behavior

- If **`ESCALATION_WORKER_INTERVAL_SECONDS`** is **not set** (or 0):
  - Worker interval = **1 hour** (unchanged from original behavior).
- No database or frontend changes; production behavior is unchanged unless overrides are set.

## Pilot / test behavior (auto-adaptive)

- If **`TEST_ESCALATION_OVERRIDE_MINUTES` > 0**:
  - Worker interval is forced to at most **30 seconds**:
    - Effective interval = **min(30, ESCALATION_WORKER_INTERVAL_SECONDS)** when `ESCALATION_WORKER_INTERVAL_SECONDS` is set and positive.
    - If `ESCALATION_WORKER_INTERVAL_SECONDS` is not set or 0, effective interval = **30 seconds**.
  - This happens **automatically at startup** (no restart later, no manual trigger).
  - Example: `TEST_ESCALATION_OVERRIDE_MINUTES=2` → worker runs every **30 seconds** → escalation can trigger in **~2 minutes** without restart or manual run.

## Exact logic used to compute interval

Computed **once at startup** in `main.go`:

1. **Default production:**  
   `intervalSeconds = 3600`, `reason = "production"`.

2. **If `TEST_ESCALATION_OVERRIDE_MINUTES > 0` (pilot override):**  
   - `intervalSeconds = 30`.  
   - If `ESCALATION_WORKER_INTERVAL_SECONDS > 0` and `< 30`:  
     `intervalSeconds = ESCALATION_WORKER_INTERVAL_SECONDS`.  
   - `reason = "pilot override"`.

3. **Else if `ESCALATION_WORKER_INTERVAL_SECONDS > 0`:**  
   - `intervalSeconds = ESCALATION_WORKER_INTERVAL_SECONDS`,  
   - `reason = "production"`.

4. Worker is created with `time.Duration(intervalSeconds) * time.Second`.

So:

- **Production, no env set:** 3600 seconds (1 hour).
- **Pilot (test override on), no env set:** 30 seconds.
- **Pilot (test override on), e.g. 10 set:** min(30, 10) = 10 seconds.
- **Production, e.g. 600 set:** 600 seconds.

## Logging (at worker startup)

On startup, the process logs exactly:

- `Escalation worker interval: X seconds`
- `Reason: production` **or** `Reason: pilot override`

(X is the computed interval in seconds.)

## Example: ~2 minute escalation (no restart, no manual trigger)

1. Set:
   - `TEST_ESCALATION_OVERRIDE_MINUTES=2`
2. (Optional) Omit `ESCALATION_WORKER_INTERVAL_SECONDS` or set it to any value ≥ 30 (e.g. 60); effective interval will be 30 seconds when test override is on.
3. Start the server.
4. Worker runs every **30 seconds**; after **~2 minutes** without status change, escalation triggers automatically.

Confirmation:

- **TEST_ESCALATION_OVERRIDE_MINUTES=2**  
  → worker runs every **30 seconds**  
  → escalation triggers automatically in **~2 minutes**.

## Files involved

- **config/config.go**  
  - `PilotConfig.EscalationWorkerIntervalSeconds` from env `ESCALATION_WORKER_INTERVAL_SECONDS` (default 0).
- **main.go**  
  - Computes `intervalSeconds` and `intervalReason` once at startup.  
  - Creates `worker.NewEscalationWorker(escalationService, interval)`.  
  - Logs `Escalation worker interval: X seconds` and `Reason: production` / `Reason: pilot override`.
- **worker/escalation_worker.go**  
  - Uses the `time.Duration` interval passed from main (no hardcoded 1h).  
  - No database or frontend changes.

## Safety

- Production behavior is unchanged unless `TEST_ESCALATION_OVERRIDE_MINUTES` or `ESCALATION_WORKER_INTERVAL_SECONDS` is set.
- No database or frontend changes.
- Interval is computed once at startup; no runtime config changes.
