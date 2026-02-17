package models

import (
	"database/sql"
	"time"
)

// ComplaintStatus represents the possible statuses of a complaint
type ComplaintStatus string

const (
	StatusDraft        ComplaintStatus = "draft"
	StatusSubmitted   ComplaintStatus = "submitted"
	StatusVerified     ComplaintStatus = "verified"
	StatusUnderReview ComplaintStatus = "under_review"
	StatusInProgress  ComplaintStatus = "in_progress"
	StatusResolved     ComplaintStatus = "resolved"
	StatusRejected     ComplaintStatus = "rejected"
	StatusClosed       ComplaintStatus = "closed"
	StatusEscalated    ComplaintStatus = "escalated"
)

// Priority represents complaint priority levels
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// ActorType represents who performed an action
type ActorType string

const (
	ActorUser    ActorType = "user"
	ActorOfficer ActorType = "officer"
	ActorSystem  ActorType = "system"
	ActorAdmin   ActorType = "admin"
)

// Complaint represents a complaint entity
type Complaint struct {
	ComplaintID          int64           `db:"complaint_id" json:"complaint_id"`
	ComplaintNumber      string          `db:"complaint_number" json:"complaint_number"`
	UserID               int64           `db:"user_id" json:"user_id"`
	Title                string          `db:"title" json:"title"`
	Description          string          `db:"description" json:"description"`
	Category             sql.NullString  `db:"category" json:"category"`
	LocationID           int64           `db:"location_id" json:"location_id"`
	Latitude             sql.NullFloat64 `db:"latitude" json:"latitude"`
	Longitude            sql.NullFloat64 `db:"longitude" json:"longitude"`
	AssignedDepartmentID sql.NullInt64   `db:"assigned_department_id" json:"assigned_department_id"`
	AssignedOfficerID    sql.NullInt64   `db:"assigned_officer_id" json:"assigned_officer_id"`
	CurrentStatus        ComplaintStatus `db:"current_status" json:"current_status"`
	Priority             Priority        `db:"priority" json:"priority"`
	IsPublic             bool            `db:"is_public" json:"is_public"`
	PublicConsentGiven   bool            `db:"public_consent_given" json:"public_consent_given"`
	SupporterCount       int             `db:"supporter_count" json:"supporter_count"`
	Pincode              sql.NullString  `db:"pincode" json:"pincode,omitempty"`
	DeviceFingerprint    sql.NullString  `db:"device_fingerprint" json:"device_fingerprint,omitempty"`
	ResolvedAt           sql.NullTime    `db:"resolved_at" json:"resolved_at"`
	ClosedAt             sql.NullTime    `db:"closed_at" json:"closed_at"`
	CreatedAt            time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt            sql.NullTime    `db:"updated_at" json:"updated_at"`
}

// StatusHistoryActorType is the audit actor type stored in complaint_status_history (system, authority, user).
type StatusHistoryActorType string

const (
	StatusHistoryActorSystem   StatusHistoryActorType = "system"
	StatusHistoryActorAuthority StatusHistoryActorType = "authority"
	StatusHistoryActorUser     StatusHistoryActorType = "user"
)

// ComplaintStatusHistory represents a status change record (immutable)
type ComplaintStatusHistory struct {
	HistoryID            int64                   `db:"history_id" json:"history_id"`
	ComplaintID          int64                   `db:"complaint_id" json:"complaint_id"`
	OldStatus            sql.NullString          `db:"old_status" json:"old_status"`
	NewStatus            ComplaintStatus         `db:"new_status" json:"new_status"`
	ChangedByType        ActorType               `db:"changed_by_type" json:"changed_by_type"`
	ChangedByUserID      sql.NullInt64           `db:"changed_by_user_id" json:"changed_by_user_id"`
	ChangedByOfficerID   sql.NullInt64           `db:"changed_by_officer_id" json:"changed_by_officer_id"`
	AssignedDepartmentID sql.NullInt64           `db:"assigned_department_id" json:"assigned_department_id"`
	AssignedOfficerID    sql.NullInt64           `db:"assigned_officer_id" json:"assigned_officer_id"`
	Notes                sql.NullString          `db:"notes" json:"notes"`
	// Audit trail: who made the change and why (all status changes must write these when possible)
	ActorType sql.NullString `db:"actor_type" json:"actor_type"` // 'system','authority','user'
	ActorID   sql.NullInt64  `db:"actor_id" json:"actor_id"`     // user_id or officer_id; NULL for system
	Reason    sql.NullString `db:"reason" json:"reason"`         // reason when available
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
}

// ComplaintAttachment represents a file attachment
type ComplaintAttachment struct {
	AttachmentID    int64          `db:"attachment_id" json:"attachment_id"`
	ComplaintID     int64          `db:"complaint_id" json:"complaint_id"`
	FileName        string         `db:"file_name" json:"file_name"`
	FilePath        string         `db:"file_path" json:"file_path"`
	FileType        sql.NullString `db:"file_type" json:"file_type"`
	FileSize        sql.NullInt64  `db:"file_size" json:"file_size"`
	UploadedByUserID sql.NullInt64 `db:"uploaded_by_user_id" json:"uploaded_by_user_id"`
	IsPublic        bool           `db:"is_public" json:"is_public"`
	CreatedAt       time.Time      `db:"created_at" json:"created_at"`
}

// ComplaintVoiceNote stores one voice note per complaint (citizen upload, not public; authority can access).
type ComplaintVoiceNote struct {
	ID               int64     `db:"id" json:"id"`
	ComplaintID      int64     `db:"complaint_id" json:"complaint_id"`
	FilePath         string    `db:"file_path" json:"file_path"`
	MimeType         string    `db:"mime_type" json:"mime_type"`
	DurationSeconds  *int      `db:"duration_seconds" json:"duration_seconds,omitempty"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

// ComplaintEvidence represents evidence integrity data for complaint attachments
type ComplaintEvidence struct {
	EvidenceID   int64          `db:"evidence_id" json:"evidence_id"`
	AttachmentID int64          `db:"attachment_id" json:"attachment_id"`
	ComplaintID  int64          `db:"complaint_id" json:"complaint_id"`
	EvidenceHash string         `db:"evidence_hash" json:"evidence_hash"`
	CapturedAt   time.Time      `db:"captured_at" json:"captured_at"`
	Latitude     sql.NullFloat64 `db:"latitude" json:"latitude,omitempty"`
	Longitude    sql.NullFloat64 `db:"longitude" json:"longitude,omitempty"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
}

// AuditLog represents an audit trail entry (immutable)
type AuditLog struct {
	AuditID         int64          `db:"audit_id" json:"audit_id"`
	EntityType      string         `db:"entity_type" json:"entity_type"`
	EntityID        int64          `db:"entity_id" json:"entity_id"`
	Action          string         `db:"action" json:"action"`
	ActionByType    ActorType      `db:"action_by_type" json:"action_by_type"`
	ActionByUserID  sql.NullInt64  `db:"action_by_user_id" json:"action_by_user_id"`
	ActionByOfficerID sql.NullInt64 `db:"action_by_officer_id" json:"action_by_officer_id"`
	OldValues       sql.NullString `db:"old_values" json:"old_values"` // JSON
	NewValues       sql.NullString `db:"new_values" json:"new_values"` // JSON
	Changes         sql.NullString `db:"changes" json:"changes"`       // JSON
	IPAddress       sql.NullString `db:"ip_address" json:"ip_address"`
	UserAgent       sql.NullString `db:"user_agent" json:"user_agent"`
	Metadata        sql.NullString `db:"metadata" json:"metadata"` // JSON
	CreatedAt       time.Time      `db:"created_at" json:"created_at"`
}

// EmailLogType is the type of authority email (assignment, escalation, resolution)
type EmailLogType string

const (
	EmailLogTypeAssignment  EmailLogType = "assignment"
	EmailLogTypeEscalation  EmailLogType = "escalation"
	EmailLogTypeResolution  EmailLogType = "resolution"
)

// EmailLog records each authority email for pilot shadow mode (all sent to pilot inbox)
// Authority abstraction: department_id + level (L1/L2/L3), not officer-based
type EmailLog struct {
	ID                   int64         `db:"id" json:"id"`
	ComplaintID          int64         `db:"complaint_id" json:"complaint_id"`
	EmailType            EmailLogType  `db:"email_type" json:"email_type"`
	IntendedAuthorityID  sql.NullInt64 `db:"intended_authority_id" json:"intended_authority_id"`
	IntendedLevel        sql.NullString `db:"intended_level" json:"intended_level"`
	DepartmentID         int64         `db:"department_id" json:"department_id"`
	SentToEmail          string        `db:"sent_to_email" json:"sent_to_email"`
	Subject              string        `db:"subject" json:"subject"`
	Body                 string        `db:"body" json:"body"`
	Status               string        `db:"status" json:"status"`                         // sent | failed
	ErrorMessage         sql.NullString `db:"error_message" json:"error_message,omitempty"` // when status=failed
	CreatedAt            time.Time     `db:"created_at" json:"created_at"`
}

// PilotMetricsEventType represents the type of pilot metrics event
type PilotMetricsEventType string

const (
	EventComplaintCreated      PilotMetricsEventType = "complaint_created"
	EventFirstAuthorityAction  PilotMetricsEventType = "first_authority_action"
	EventEscalationTriggered   PilotMetricsEventType = "escalation_triggered"
	EventComplaintResolved     PilotMetricsEventType = "complaint_resolved"
	EventChatAbandoned         PilotMetricsEventType = "chat_abandoned"
)

// PilotMetricsEvent represents a pilot metrics event
type PilotMetricsEvent struct {
	ID          int64                  `db:"id" json:"id"`
	EventType   PilotMetricsEventType  `db:"event_type" json:"event_type"`
	ComplaintID sql.NullInt64          `db:"complaint_id" json:"complaint_id"`
	UserID      sql.NullInt64          `db:"user_id" json:"user_id"`
	Metadata    sql.NullString         `db:"metadata" json:"metadata"` // JSON string
	CreatedAt   time.Time              `db:"created_at" json:"created_at"`
}
