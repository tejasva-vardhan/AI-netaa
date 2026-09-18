# AI Neta

A civic-complaint app: residents file issues in a chat UI, officers work a queue, and a background worker raises the ticket if the SLA is missed.

Citizens describe the problem, attach photo / voice / GPS, and get a tracking number. The backend stores the ticket in MySQL, assigns a department from **category rules** (not a language model), and exposes a public case page with no personal fields. Officers log in separately, change status with a reason, and see only assigned tickets.

Live UI: https://ai-netaa.vercel.app

Phone OTP is a local demo (in-memory code, no SMS gateway). Escalation is time-based SQL rules, not an LLM judge.

## How a complaint moves

```
Chat / camera / GPS
        │
        ▼
  Go API  ── MySQL ──  officer dashboard
        │
        ├── keyword category → department (PWD, water, electricity, …)
        ├── JWT citizen auth; separate authority JWT
        └── escalation worker (L1 → L2 at 72h, L2 → L3 at 120h)
```

1. **Intake** — `ChatScreen` plus optional `CameraScreen`, `LocationScreen`, `PhoneVerificationScreen`.
2. **Category** — `frontend/src/utils/categoryInference.js` (regex / keywords). Backend `department_repository.go` maps category → department id, then location / collector fallback.
3. **Create** — `POST /api/v1/complaints`. The API currently requires photo and GPS even if the UI can skip those steps.
4. **Citizen follow-up** — list, detail, status timeline, voice upload.
5. **Public** — `/case/:complaintNumber` and `GET /api/v1/public/complaints/by-number/{n}` omit phone, GPS, and officer notes.
6. **Officer** — `/authority/login`, assigned queue, `POST .../status` with a reason. Notes: `POST .../note` exists on the API only (no UI yet).
7. **SLA** — `worker/escalation_worker.go` ticks on an interval, applies `seed_escalation_rules_sla.sql`. Email goes through shadow mode (one inbox) so the pilot does not mail real departments.

## Features

- Chat-first filing with photo, voice, GPS
- Offline retry queue in the frontend (`utils/offlineQueue.js`)
- Officer dashboard (login, queue, status + reason)
- Public case page
- Escalation L1 / L2 / L3
- Health check at `/health`

## Stack

| Layer | Tech |
|---|---|
| API | Go, Gorilla mux, JWT |
| DB | MySQL |
| Workers | Go goroutines (`worker/`) |
| Frontend | React 18, Vite, React Router |
| Hosting | API anywhere that runs Go; UI on Vercel |

## HTTP (`/api/v1`)

**Citizens**

- `POST /users/otp/send`, `POST /users/otp/verify`
- `GET/POST /complaints`, `GET /complaints/{id}`, `GET /complaints/{id}/timeline`
- `POST /complaints/{id}/voice`

**Officers** (`/authority`)

- `POST /login`, `GET /me`, `GET /complaints`
- `POST /complaints/{id}/status`, `POST /complaints/{id}/note`

**Public**

- `GET /public/complaints/by-number/{complaint_number}`

## Layout

```
main.go                 process entry (DB, workers, HTTP)
handler/ service/ repository/ routes/ models/
worker/                 escalation + notifications
migrations/             0001–0006
seed_*.sql              pilot officers and SLA hours
frontend/src/screens/   citizen + officer + public UI
frontend/src/pages/     login / signup / tracker (older routes still mounted)
```

## Run locally

Go 1.21+, Node 18+, MySQL 5.7+ or MariaDB.

```bash
git clone https://github.com/tejasva-vardhan/AI-netaa.git
cd AI-netaa
go mod download
```

`.env`:

```
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=finalneta
SERVER_PORT=8080
JWT_SECRET=change-me
```

Apply `migrations/0001` … `0006`, then the `seed_*.sql` files for the pilot officers and SLA rules.

```bash
go run .                 # API :8080
cd frontend && npm install && npm run dev
```

`frontend/.env`:

```
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

Do not commit `.env` files.

## License

MIT.
