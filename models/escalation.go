package models

import (
	"database/sql"
	"time"
)

// EscalationRule represents an escalation rule from escalation_rules table
type EscalationRule struct {
	RuleID            int64          `db:"rule_id" json:"rule_id"`
	FromDepartmentID  sql.NullInt64  `db:"from_department_id" json:"from_department_id"`
	FromLocationID    sql.NullInt64  `db:"from_location_id" json:"from_location_id"`
	ToDepartmentID   sql.NullInt64  `db:"to_department_id" json:"to_department_id"` // NULL = escalate within same department hierarchy
	ToLocationID      sql.NullInt64 `db:"to_location_id" json:"to_location_id"`
	EscalationLevel   int            `db:"escalation_level" json:"escalation_level"`
	Conditions        sql.NullString `db:"conditions" json:"conditions"` // JSON
	IsActive          bool           `db:"is_active" json:"is_active"`
	CreatedAt         time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt         sql.NullTime   `db:"updated_at" json:"updated_at"`
}

// EscalationConditions represents the JSON conditions for escalation rules
type EscalationConditions struct {
	// StatusConditions - Escalate if complaint is in these statuses
	Statuses []string `json:"statuses,omitempty"`
	
	// TimeBasedConditions - Escalate based on time since last update
	TimeBased *TimeBasedCondition `json:"time_based,omitempty"`
	
	// PriorityConditions - Escalate based on priority
	Priorities []string `json:"priorities,omitempty"`
	
	// ReminderConditions - Send reminder instead of escalation
	IsReminder bool `json:"is_reminder,omitempty"`
	
	// ReminderIntervalHours - Hours between reminders (for reminders only)
	ReminderIntervalHours *int `json:"reminder_interval_hours,omitempty"`
}

// TimeBasedCondition represents time-based escalation conditions
type TimeBasedCondition struct {
	// HoursSinceLastUpdate - Escalate if last update was X hours ago
	HoursSinceLastUpdate int `json:"hours_since_last_update"`
	
	// SLAHours - SLA hours for escalation (clear meaning: hours since status change)
	SLAHours int `json:"sla_hours,omitempty"`
	
	// HoursSinceStatusChange - Legacy field name (deprecated, use sla_hours)
	HoursSinceStatusChange int `json:"hours_since_status_change,omitempty"`
	
	// HoursSinceCreation - Escalate if complaint created X hours ago
	HoursSinceCreation int `json:"hours_since_creation,omitempty"`
}

// ComplaintEscalation represents an escalation record
type ComplaintEscalation struct {
	EscalationID        int64          `db:"escalation_id" json:"escalation_id"`
	ComplaintID         int64          `db:"complaint_id" json:"complaint_id"`
	FromDepartmentID    sql.NullInt64  `db:"from_department_id" json:"from_department_id"`
	FromOfficerID       sql.NullInt64  `db:"from_officer_id" json:"from_officer_id"`
	ToDepartmentID      int64          `db:"to_department_id" json:"to_department_id"`
	ToOfficerID         sql.NullInt64  `db:"to_officer_id" json:"to_officer_id"`
	EscalationLevel     int            `db:"escalation_level" json:"escalation_level"`
	Reason              sql.NullString `db:"reason" json:"reason"`
	EscalatedByType     ActorType      `db:"escalated_by_type" json:"escalated_by_type"`
	EscalatedByUserID   sql.NullInt64  `db:"escalated_by_user_id" json:"escalated_by_user_id"`
	EscalatedByOfficerID sql.NullInt64 `db:"escalated_by_officer_id" json:"escalated_by_officer_id"`
	StatusHistoryID     sql.NullInt64  `db:"status_history_id" json:"status_history_id"`
	CreatedAt           time.Time      `db:"created_at" json:"created_at"`
}

// EscalationCandidate represents a complaint that may need escalation
type EscalationCandidate struct {
	ComplaintID          int64
	ComplaintNumber      string
	CurrentStatus        ComplaintStatus
	Priority             Priority
	AssignedDepartmentID sql.NullInt64
	AssignedOfficerID    sql.NullInt64
	LocationID           int64
	Pincode              sql.NullString // Pincode for authority lookup
	CreatedAt            time.Time
	UpdatedAt            sql.NullTime
	LastStatusChangeAt   time.Time // From status history
}

// EscalationResult represents the result of escalation processing
type EscalationResult struct {
	ComplaintID      int64     `json:"complaint_id"`
	Escalated        bool      `json:"escalated"`
	EscalationID     *int64    `json:"escalation_id,omitempty"`
	NewStatus        *string   `json:"new_status,omitempty"`
	Reason           string    `json:"reason"`
	ProcessedAt      time.Time `json:"processed_at"`
}

// ReminderResult represents the result of reminder processing
type ReminderResult struct {
	ComplaintID     int64     `json:"complaint_id"`
	ReminderSent    bool      `json:"reminder_sent"`
	Reason          string    `json:"reason"`
	ProcessedAt     time.Time `json:"processed_at"`
}
