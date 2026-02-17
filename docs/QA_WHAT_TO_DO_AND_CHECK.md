# What to Do and Check (Manual QA)

Processes on ports **8080** and **5173** have been stopped. Follow this to run the app and check the UI yourself.

---

## 1. Start services

**Terminal 1 – MySQL**   
- Ensure MySQL is running (e.g. start from Services or `net start MySQL80` if needed).

**Terminal 2 – Backend**
```bash
cd c:\Users\tejas\OneDrive\Desktop\finalneta
set ADMIN_TOKEN=pilot-admin-qa
set TEST_ESCALATION_OVERRIDE_MINUTES=1
go run .
```

**Terminal 3 – Citizen frontend**
```bash
cd c:\Users\tejas\OneDrive\Desktop\finalneta\frontend
npm run dev
```

**Terminal 4 (optional) – Authority frontend**  
- If you have a separate authority app, start it and note its URL (e.g. another port or path).

---

## 2. What to check in the browser

### Citizen app (e.g. http://localhost:5173)

1. **Login**
   - Open the app → click **“Talk with me”**.
   - Enter phone (e.g. 10 digits) → get OTP → enter OTP → confirm you’re logged in.
   - Refresh the page → you should still be logged in.

2. **New complaint**
   - Start a new chat/conversation.
   - Enter complaint text.
   - Allow **GPS** (live location only; no manual lat/long).
   - Allow **Camera** (live photo only; no gallery).
   - Submit → see success message.
   - Open **“My Complaints”** → your complaint is listed.
   - Refresh → complaint still there.

### Authority app (if separate)

3. **Authority**
   - Open authority URL (e.g. authority frontend or same app with /authority).
   - **Without login** → you should be blocked (redirect or 401).
   - Log in as authority (e.g. `qa.officer@pilot.test` / `TestPass123`).
   - Open dashboard → only **assigned** complaints should appear.
   - Open an **escalated** complaint → change status to **under_review** with a reason → save.
   - Try saving **without** reason → should fail.
   - Save **with** reason → should succeed. Refresh → status should stay updated.

### Public case page

4. **Public case**
   - Copy a **complaint_number** (e.g. `COMP-20260214-6eef7216`) from My Complaints or DB.
   - Open in browser: **http://localhost:5173/case/COMP-20260214-6eef7216** (or your app’s public case route).
   - Check: timeline and status visible; **no** PII, **no** GPS, **no** complaint_id, **no** images.

### Email (shadow inbox)

5. **Inbox**
   - Log in to **aineta502@gmail.com**.
   - After submitting a complaint: check for **assignment** email (dashboard link, no action links).
   - After an escalation (wait ~1 min with TEST_ESCALATION_OVERRIDE_MINUTES=1 or trigger via API): check for **escalation** email.
   - After authority updates status to resolved/closed: check for **resolution** email and that dashboard link works (login required).

---

## 3. If something fails

- **Backend 500 / DB errors:** Ensure migrations are applied (`migrations/0002_*, 0003_*, 0004_*`) and `email_logs` table exists (`database_email_logs.sql`).
- **“Cannot change from escalated”:** Backend must be restarted after the latest code change (authority can move escalated → under_review / in_progress).
- **Escalation never runs:** Confirm `TEST_ESCALATION_OVERRIDE_MINUTES=1` and complaint is **verified** and has **assigned_department_id**; wait at least 1 minute and check `complaint_escalations` in DB.

Full API/DB results and fixes are in **QA_RUN_REPORT.md**.
