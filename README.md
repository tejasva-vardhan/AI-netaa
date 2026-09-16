# AI Neta

Student project: a chat UI for civic complaints. A citizen files a complaint; the backend stores it, assigns a department, and raises the ticket L0 to L3 if the SLA is missed.

Stack: Go API, React frontend, MySQL. Phone OTP for citizens. Officers get a dashboard for assigned tickets. Escalation is time-based, not a language-model judge.

This is coursework/personal software, not a deployed city system.

## What it does

- Citizen: chat form, optional photo/voice/GPS, status timeline, public case page (no personal data on the public page)
- Officer: login, assigned queue, status updates with a reason, internal notes
- System: L0 to L3 escalation worker, email log (shadow mode can send all mail to one inbox)

## Layout

- `backend/` — Go HTTP API, workers, MySQL migrations
- `frontend/` — React + Vite
- `docs/` — extra notes

## Run locally

Needs Go 1.21+, Node 18+, MySQL 5.7+ (or MariaDB).

### Backend

```bash
git clone https://github.com/tejasva-vardhan/AI-netaa.git
cd AI-netaa
go mod download
```

Create `.env` with at least:

```
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=finalneta
SERVER_PORT=8080
JWT_SECRET=change-me
```

Apply SQL files in `migrations/` in order (`0001` through `0006`), then:

```bash
go run .
```

API: http://localhost:8080

### Frontend

```bash
cd frontend
npm install
```

Create `frontend/.env`:

```
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

```bash
npm run dev
```

UI: http://localhost:3000 (or the next free port)

Do not commit `.env` files.

## Author

Tejasva Vardhan Sharma
