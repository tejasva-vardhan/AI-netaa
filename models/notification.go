package models

import (
	"database/sql"
	"time"
)

// NotificationChannel represents the notification channel type
type NotificationChannel string

const (
	ChannelEmail    NotificationChannel = "email"
	ChannelSMS      NotificationChannel = "sms"
	ChannelWhatsApp NotificationChannel = "whatsapp"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
	NotificationStatusRetrying NotificationStatus = "retrying"
)

// NotificationPriority represents notification priority
type NotificationPriority string

const (
	NotificationPriorityLow    NotificationPriority = "low"
	NotificationPriorityNormal NotificationPriority = "normal"
	NotificationPriorityHigh   NotificationPriority = "high"
	NotificationPriorityUrgent NotificationPriority = "urgent"
)

// Notification represents a notification record
type Notification struct {
	NotificationID   int64              `db:"notification_id" json:"notification_id"`
	EntityType       string             `db:"entity_type" json:"entity_type"` // e.g., "complaint", "user"
	EntityID         int64              `db:"entity_id" json:"entity_id"`
	Channel          NotificationChannel `db:"channel" json:"channel"`
	Recipient        string             `db:"recipient" json:"recipient"` // email, phone number, etc.
	Subject          sql.NullString     `db:"subject" json:"subject"`
	Body             string             `db:"body" json:"body"`
	TemplateID       sql.NullString     `db:"template_id" json:"template_id"`
	TemplateData     sql.NullString     `db:"template_data" json:"template_data"` // JSON
	Status           NotificationStatus  `db:"status" json:"status"`
	Priority         NotificationPriority `db:"priority" json:"priority"`
	RetryCount       int                `db:"retry_count" json:"retry_count"`
	MaxRetries       int                `db:"max_retries" json:"max_retries"`
	NextRetryAt      sql.NullTime        `db:"next_retry_at" json:"next_retry_at"`
	SentAt           sql.NullTime        `db:"sent_at" json:"sent_at"`
	FailedAt         sql.NullTime        `db:"failed_at" json:"failed_at"`
	ErrorMessage     sql.NullString      `db:"error_message" json:"error_message"`
	CreatedAt        time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt        sql.NullTime        `db:"updated_at" json:"updated_at"`
}

// NotificationLog represents a log entry for notification attempts
type NotificationLog struct {
	LogID          int64              `db:"log_id" json:"log_id"`
	NotificationID int64              `db:"notification_id" json:"notification_id"`
	AttemptNumber  int                `db:"attempt_number" json:"attempt_number"`
	Status         NotificationStatus  `db:"status" json:"status"`
	ErrorMessage   sql.NullString     `db:"error_message" json:"error_message"`
	ResponseData   sql.NullString     `db:"response_data" json:"response_data"` // JSON
	CreatedAt      time.Time          `db:"created_at" json:"created_at"`
}

// NotificationRequest represents a request to send a notification
type NotificationRequest struct {
	EntityType   string              `json:"entity_type"`
	EntityID     int64               `json:"entity_id"`
	Channel      NotificationChannel `json:"channel"`
	Recipient    string              `json:"recipient"`
	Subject      *string             `json:"subject,omitempty"`
	Body         string              `json:"body"`
	TemplateID   *string             `json:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`
	Priority     NotificationPriority `json:"priority,omitempty"`
	MaxRetries   *int                `json:"max_retries,omitempty"`
}

// NotificationResult represents the result of sending a notification
type NotificationResult struct {
	NotificationID int64             `json:"notification_id"`
	Status          NotificationStatus `json:"status"`
	Success         bool              `json:"success"`
	Message         string            `json:"message"`
	SentAt          *time.Time        `json:"sent_at,omitempty"`
}

// NotificationConfig holds configuration for notification system
type NotificationConfig struct {
	// Default retry configuration
	DefaultMaxRetries int
	
	// Retry backoff configuration
	InitialRetryDelay time.Duration
	MaxRetryDelay     time.Duration
	BackoffMultiplier float64
	
	// Worker configuration
	WorkerBatchSize   int
	WorkerInterval    time.Duration
	
	// Queue configuration
	QueueMaxSize      int
}

// DefaultNotificationConfig returns default notification configuration
func DefaultNotificationConfig() *NotificationConfig {
	return &NotificationConfig{
		DefaultMaxRetries: 3,
		InitialRetryDelay: 1 * time.Minute,
		MaxRetryDelay:     30 * time.Minute,
		BackoffMultiplier: 2.0,
		WorkerBatchSize:   100,
		WorkerInterval:    30 * time.Second,
		QueueMaxSize:      10000,
	}
}
