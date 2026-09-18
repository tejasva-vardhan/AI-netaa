# AI Neta frontend

React 18 + Vite client for the Go complaint API (`VITE_API_BASE_URL`, e.g. `http://localhost:8080/api/v1`).

## Citizen flow

| Screen | What it does |
|---|---|
| `LandingScreen` | start filing |
| `ChatScreen` | text intake, category hints |
| `CameraScreen` | photo |
| `LocationScreen` | GPS |
| `PhoneVerificationScreen` | demo OTP |
| `ComplaintsListScreen` / `ComplaintDetailScreen` | personal tickets + timeline |
| `PublicCaseScreen` | `/case/:complaintNumber` with no PII |

Auth store: `stores/authStore.js`. Draft / offline retries: `utils/offlineQueue.js`. Category keywords: `utils/categoryInference.js`.

## Officer flow

`/authority/login` → dashboard queue → complaint detail (status + reason). Notes API is not wired in this UI.

```bash
npm install
npm run dev
```
