# AI Neta

Student project: a chat-style civic complaint system. A resident describes a problem, the backend stores a ticket, assigns a department by **category rules**, and a worker raises the ticket if nobody updates it in time.

Routing is not an LLM. Escalation is not an LLM. Phone OTP is a **local demo** (in-memory code, no SMS provider). This is coursework, not a deployed city system.

Demo UI: https://ai-netaa.vercel.app

## What a ticket does

**Citizen (React chat)**

1. Describe the issue in chat. Optional photo, voice note, and GPS on the filing screens.
2. The UI infers a category (infrastructure, water, electricity, sanitation, health, …) with keyword rules in `frontend/src/utils/categoryInference.js`.
3. `POST` creates the complaint in MySQL. The handler currently requires photo and GPS even if the UI lets you skip them.
4. Phone OTP (`POST /api/v1/users/otp/send` and `/verify`) is demo-only; the server can return `debug_otp`.
5. The citizen can watch a status timeline. A **public** page `/case/:complaintNumber` shows the case without phone, GPS, or officer notes.

**Officer dashboard**

- Login, assigned queue, status change with a mandatory reason.
- Internal notes exist as `POST /authority/complaints/{id}/note` on the API; there is no notes control in the officer UI yet.

**SLA worker** (`worker/escalation_worker.go`)

- Runs on a timer, not on each HTTP request.
- Levels in SQL are **L1 → L2 → L3** (72 hours then 120 hours in `seed_escalation_rules_sla.sql`).
- If the ticket is not updated, the worker raises the level. Email is logged in shadow mode (can send everything to one inbox).

**Department assignment** (`repository/department_repository.go`)

Category switch, for example infrastructure → PWD, water → water board, electricity → electricity dept, then location / collector fallback. Not model-based routing.

## Stack

- API: Go (`main.go` at repo root), Gorilla mux, MySQL
- UI: React 18, Vite (`frontend/`)
- Background: escalation worker + notification worker
- Auth: JWT for citizens; separate authority login for officers

## Layout

```
main.go, handler/, service/, repository/, routes/
worker/           Escalation and notification loops
migrations/       Ordered SQL (0001–0006)
seed_*.sql        Pilot officers and SLA rules
frontend/         Chat, camera, location, officer screens, public case page
```

## HTTP (prefix `/api/v1`)

- Citizens: complaints CRUD, timeline, voice upload, OTP
- Officers: `/authority/login`, `/authority/complaints`, status update
- Public: `/public/complaints/by-number/{complaint_number}`
- Health: `/health`

## Run locally

Needs Go 1.21+, Node 18+, MySQL 5.7+ (or MariaDB).

```bash
git clone https://github.com/tejasva-vardhan/AI-netaa.git
cd AI-netaa
go mod download
```

Create `.env`:

```
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=finalneta
SERVER_PORT=8080
JWT_SECRET=change-me
```

Apply `migrations/0001` through `0006`, then seed files if you want the pilot officers and SLA rules.

```bash
go run .
```

API: http://localhost:8080

```bash
cd frontend
npm install
```

`frontend/.env`:

```
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

```bash
npm run dev
```

Do not commit `.env` files.

## Author

Tejasva Vardhan Sharma
