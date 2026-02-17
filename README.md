# AI Neta - Public Accountability System

A production-grade complaint management system with AI-powered chat interface, voice notes, escalation workflows, and email notifications. Built with Go backend and React frontend.

## 🎯 Overview

AI Neta enables citizens to file complaints through an intuitive chat interface, track their status, and receive updates. Authorities can manage assigned complaints, update statuses, and escalate issues when needed. The system includes automated escalation, email notifications (shadow mode for pilot), and public case pages for transparency.

## ✨ Features

### Citizen Features
- **Chat-based complaint filing** - Natural language conversation with AI assistant
- **Phone OTP authentication** - Secure login via phone number verification
- **Live photo capture** - Camera-based evidence capture (gallery option available)
- **Voice notes** - Optional voice recording attached to complaints
- **GPS location** - Automatic location capture for complaints
- **Status tracking** - Real-time complaint status and timeline
- **Public case pages** - Shareable complaint pages (no PII exposed)

### Authority Features
- **Dashboard** - View only assigned complaints
- **Status management** - Update complaint status with mandatory reasons
- **Internal notes** - Add notes visible to authority users
- **Escalation handling** - Manage escalated complaints

### System Features
- **Automated escalation** - Time-based escalation with configurable SLAs
- **Email notifications** - Shadow mode (all emails to pilot inbox) or production SendGrid
- **Audit trail** - Complete status history with actor tracking
- **Abuse prevention** - Rate limiting and duplicate detection
- **Public API** - Shareable case pages by complaint number

## 🏗️ Architecture

```
finalneta/
├── backend/
│   ├── main.go              # Application entry point
│   ├── config/              # Configuration management
│   ├── handler/             # HTTP request handlers
│   ├── service/             # Business logic layer
│   ├── repository/          # Database access layer
│   ├── models/              # Data models (entities, DTOs)
│   ├── routes/              # Route configuration
│   ├── middleware/          # Auth, CORS middleware
│   ├── worker/              # Background workers (escalation, notifications)
│   ├── notification/        # Email sender (SendGrid support)
│   ├── migrations/          # Database migrations
│   └── cmd/                 # CLI tools (verify_escalation)
├── frontend/
│   ├── src/
│   │   ├── pages/           # Main pages (Chat, Dashboard, etc.)
│   │   ├── screens/         # Screen components
│   │   ├── components/      # Reusable components
│   │   ├── stores/          # Zustand state management
│   │   ├── services/        # API client
│   │   └── utils/           # Utilities
│   └── package.json
└── docs/                    # Documentation
```

## 🚀 Quick Start

### Prerequisites

- **Go** 1.21+ (backend)
- **Node.js** 18+ and npm (frontend)
- **MySQL** 5.7+ or MariaDB 10.3+
- **Git**

### Backend Setup

1. **Clone repository**
```bash
git clone <repository-url>
cd finalneta
```

2. **Install Go dependencies**
```bash
go mod download
```

3. **Configure environment** (create `.env` file)
```bash
# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=finalneta

# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# JWT
JWT_SECRET=your-secret-key-change-in-production

# Email (optional - for production)
EMAIL_MODE=shadow                    # shadow | production
EMAIL_SHADOW_ADDRESS=aineta502@gmail.com
SENDGRID_API_KEY=your-sendgrid-key   # Optional: for real email delivery
SENDGRID_FROM_EMAIL=noreply@aineta.in
SENDGRID_FROM_NAME=AI Neta

# Pilot/Testing
ADMIN_TOKEN=pilot-admin-qa           # For admin endpoints
TEST_ESCALATION_OVERRIDE_MINUTES=1  # Override escalation SLA for testing
ESCALATION_WORKER_INTERVAL_SECONDS=30

# Frontend URL (for email links)
FRONTEND_URL=http://localhost:3000

# Uploads
UPLOAD_BASE_PATH=uploads            # Default: uploads/
```

4. **Run database migrations**
```bash
# Apply migrations in order:
mysql -u root -p finalneta < migrations/0001_complaint_status_history_audit_columns.sql
mysql -u root -p finalneta < migrations/0002_authority_pilot_tables.sql
mysql -u root -p finalneta < migrations/0003_officers_authority_level.sql
mysql -u root -p finalneta < migrations/0004_add_verified_status_enum.sql
mysql -u root -p finalneta < migrations/0005_complaint_voice_notes.sql
mysql -u root -p finalneta < migrations/0006_email_logs_status.sql
```

5. **Start backend**
```bash
go run .
```

Backend runs on `http://localhost:8080`

### Frontend Setup

1. **Navigate to frontend directory**
```bash
cd frontend
```

2. **Install dependencies**
```bash
npm install
```

3. **Configure environment** (create `frontend/.env`)
```bash
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

4. **Start development server**
```bash
npm run dev
```

Frontend runs on `http://localhost:3000` (or next available port)

## 📡 API Endpoints

### Authentication

**POST** `/api/v1/users/otp/send`
- Send OTP to phone number
- Body: `{ "phone_number": "9876543210" }`
- Response: `{ "success": true, "otp": "123456" }` (dev mode includes OTP)

**POST** `/api/v1/users/otp/verify`
- Verify OTP and get JWT token
- Body: `{ "phone_number": "9876543210", "otp": "123456" }`
- Response: `{ "success": true, "token": "jwt...", "user_id": 1 }`

### Complaints (Citizen - Requires Auth)

**POST** `/api/v1/complaints`
- Create new complaint
- Headers: `Authorization: Bearer <token>`
- Body: `{ "title": "...", "description": "...", "location_id": 1, "latitude": 28.6, "longitude": 77.2, "attachment_urls": ["url"], ... }`
- Response: `{ "complaint_id": 1, "complaint_number": "COMP-20260215-abc123", "status": "submitted" }`

**GET** `/api/v1/complaints`
- Get user's complaints list
- Headers: `Authorization: Bearer <token>`

**GET** `/api/v1/complaints/{id}`
- Get complaint details (owner only)
- Headers: `Authorization: Bearer <token>`

**GET** `/api/v1/complaints/{id}/timeline`
- Get status timeline
- Headers: `Authorization: Bearer <token>`

**POST** `/api/v1/complaints/{id}/voice`
- Upload voice note (owner only, overwrites existing)
- Headers: `Authorization: Bearer <token>`, `Content-Type: audio/webm` or `audio/wav`
- Body: Raw audio blob
- Response: `{ "message": "Voice note attached", "complaint_id": 1 }`

### Authority (Requires Authority Auth)

**POST** `/api/v1/authority/login`
- Authority login
- Body: `{ "email": "officer@example.com", "password": "password" }`
- Response: `{ "success": true, "token": "jwt...", "officer_id": 1 }`

**GET** `/api/v1/authority/complaints`
- Get assigned complaints
- Headers: `Authorization: Bearer <authority_token>`

**POST** `/api/v1/authority/complaints/{id}/status`
- Update complaint status
- Headers: `Authorization: Bearer <authority_token>`
- Body: `{ "new_status": "under_review", "reason": "..." }` (reason required)

**POST** `/api/v1/authority/complaints/{id}/note`
- Add internal note
- Headers: `Authorization: Bearer <authority_token>`
- Body: `{ "note_text": "...", "is_visible_to_citizen": false }`

### Public

**GET** `/api/v1/public/complaints/by-number/{complaint_number}`
- Get public case page (no auth required)
- Response: `{ "complaint_number": "...", "current_status": "...", "timeline": [...] }`
- No PII, GPS, or images exposed

### Admin

**POST** `/api/v1/complaints/{id}/verify`
- Verify complaint (admin only)
- Headers: `X-Admin-Token: <ADMIN_TOKEN>`

**POST** `/api/v1/escalations/process`
- Trigger escalation worker manually (admin only)
- Headers: `X-Admin-Token: <ADMIN_TOKEN>`

## 🔐 Authentication

### Citizen Authentication
- Phone number + OTP verification
- JWT token stored in `localStorage` (`auth_token`)
- Token includes `user_id` and `actor_type: "user"`
- Middleware validates token and sets `user_id` in context

### Authority Authentication
- Email + password login
- JWT token includes `officer_id`, `authority_level`, `actor_type: "authority"`
- Middleware validates token and sets `officer_id` in context

### Security
- Citizen tokens rejected on authority endpoints
- Authority tokens rejected on citizen endpoints
- CORS configured for frontend origin
- No PATCH endpoints exposed (only POST for status updates)

## 📧 Email System

### Shadow Mode (Pilot)
- **All emails sent to**: `aineta502@gmail.com` (configurable via `EMAIL_SHADOW_ADDRESS`)
- **Email types**: Assignment, Escalation, Resolution
- **Logging**: All emails logged to `email_logs` table with status (`sent`/`failed`) and error messages
- **Non-blocking**: Email failures never break complaint flows

### Production Mode
- Set `EMAIL_MODE=production` and `SENDGRID_API_KEY`
- Emails sent via SendGrid API
- Still respects shadow address if `EMAIL_MODE=shadow`

### Email Triggers
- **Assignment**: When complaint is assigned to department
- **Escalation**: When complaint escalates to next level
- **Resolution**: When authority updates status to resolved/closed

## 🗄️ Database

### Key Tables
- `complaints` - Main complaint records
- `complaint_status_history` - Status change audit trail
- `complaint_voice_notes` - Voice attachments (one per complaint)
- `complaint_attachments` - Photo attachments
- `email_logs` - Email delivery logs
- `complaint_escalations` - Escalation records
- `users` - Citizen users (phone-based)
- `officers` - Authority officers
- `authority_credentials` - Officer login credentials

### Migrations
Run migrations in order (`0001_*.sql` through `0006_*.sql`). See `migrations/` directory.

## 🔄 Escalation System

- **Automatic escalation** based on SLA (time since status change)
- **Escalation levels**: L0 → L1 → L2 → L3
- **Rules**: Configurable per department/location
- **Worker**: Runs every 30 seconds (configurable)
- **Testing**: Use `TEST_ESCALATION_OVERRIDE_MINUTES=1` for 1-minute SLA override

### Escalation CLI
```bash
go run ./cmd/verify_escalation
```
Runs one escalation cycle and reports results.

## 🧪 Testing

### Manual QA
See `docs/QA_WHAT_TO_DO_AND_CHECK.md` for manual testing steps.

### Test Environment Variables
```bash
ADMIN_TOKEN=pilot-admin-qa
TEST_ESCALATION_OVERRIDE_MINUTES=1
ESCALATION_WORKER_INTERVAL_SECONDS=15
```

### Example API Calls

**Create complaint (after OTP auth)**
```bash
curl -X POST http://localhost:8080/api/v1/complaints \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Road pothole",
    "description": "Large pothole on Main Street",
    "location_id": 1,
    "latitude": 28.6139,
    "longitude": 77.2090,
    "attachment_urls": ["data:image/jpeg;base64,..."],
    "priority": "medium",
    "public_consent_given": true
  }'
```

**Get public case**
```bash
curl http://localhost:8080/api/v1/public/complaints/by-number/COMP-20260215-abc123
```

## 🐛 Troubleshooting

### Backend won't start
- Check MySQL is running
- Verify `.env` file exists with correct DB credentials
- Ensure migrations are applied

### Frontend shows "Complaint submit nahi ho payi"
- Check backend is running on port 8080
- Verify `VITE_API_BASE_URL` in `frontend/.env`
- Check browser console for API errors
- Ensure phone is verified (OTP flow completed)

### Emails not sending
- Check `email_logs` table for status (`sent`/`failed`)
- Verify `EMAIL_SHADOW_ADDRESS` is set
- If using SendGrid, verify `SENDGRID_API_KEY` is valid

### Escalation not working
- Ensure `TEST_ESCALATION_OVERRIDE_MINUTES=1` for testing
- Check complaint has `assigned_department_id`
- Verify complaint status is eligible (verified, under_review, in_progress)
- Check `complaint_escalations` table for records

## 📝 Development

### Project Structure
- **Backend**: Go modules, clean architecture (handler → service → repository)
- **Frontend**: React + Vite, Zustand for state, React Router
- **State Management**: Zustand stores (`chatStore`, `authStore`)
- **API Client**: Centralized in `frontend/src/services/api.js`

### Code Style
- Go: Standard formatting (`go fmt`)
- JavaScript: ESLint configured
- Commits: Conventional commits preferred

## 📄 License

[Your License Here]

## 🤝 Contributing

[Contributing guidelines]

## 📞 Support

For issues and questions, see `docs/` directory for detailed documentation.
