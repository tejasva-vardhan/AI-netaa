# AI Neta

Student project: a chat UI for civic complaints. A citizen files a ticket; the backend stores it, assigns a department, and raises the ticket if the SLA is missed.

Stack: Go API at the repository root, React frontend, MySQL. Escalation is a time-based worker, not a language-model judge. Phone OTP is a local demo (no SMS provider).

This is coursework, not a deployed city system.

## What it does

- Citizen: chat form, photo, voice, GPS, status timeline, and a public case page with no personal data
- Officer: login, assigned queue, and status updates with a reason
- System: department routing and L1–L3 SLA escalation

## Layout

- Repository root — Go HTTP API, workers, MySQL schema
- `frontend/` — React + Vite
- `migrations/` — SQL migrations

## Run locally

Needs Go 1.21+, Node 18+, MySQL 5.7+ (or MariaDB).

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

Do not commit `.env` files.

## Author

Tejasva Vardhan Sharma
