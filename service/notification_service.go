package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"finalneta/models"
	"finalneta/notification"
	"finalneta/repository"
	"fmt"
	"math"
	"time"
)

// NotificationService handles notification sending with retry logic
type NotificationService struct {
	repo           *repository.NotificationRepository
	complaintRepo  *repository.ComplaintRepository
	senders        map[models.NotificationChannel]notification.Sender
	config         *models.NotificationConfig
}

// NotificationConfig is an alias for models.NotificationConfig
type NotificationConfig = models.NotificationConfig

// NewNotificationService creates a new notification service
func NewNotificationService(
	repo *repository.NotificationRepository,
	complaintRepo *repository.ComplaintRepository,
	config *models.NotificationConfig,
) *NotificationService {
	if config == nil {
		config = models.DefaultNotificationConfig()
	}

	// Initialize senders for each channel
	senders := make(map[models.NotificationChannel]notification.Sender)
	senders[models.ChannelEmail] = notification.NewEmailSender()
	senders[models.ChannelSMS] = notification.NewSMSSender()
	senders[models.ChannelWhatsApp] = notification.NewWhatsAppSender()

	return &NotificationService{
		repo:          repo,
		complaintRepo: complaintRepo,
		senders:       senders,
		config:        config,
	}
}

// QueueNotification queues a notification for async sending
// This method is non-blocking and returns immediately
// The notification will be processed by the background worker
func (s *NotificationService) QueueNotification(req *models.NotificationRequest) (*models.NotificationResult, error) {
	// Set defaults
	maxRetries := s.config.DefaultMaxRetries
	if req.MaxRetries != nil {
		maxRetries = *req.MaxRetries
	}

	priority := models.NotificationPriorityNormal
	if req.Priority != "" {
		priority = req.Priority
	}

	// Serialize template data if provided
	var templateDataJSON sql.NullString
	if len(req.TemplateData) > 0 {
		data, err := json.Marshal(req.TemplateData)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize template data: %w", err)
		}
		templateDataJSON = sql.NullString{String: string(data), Valid: true}
	}

	// Create notification record
	notification := &models.Notification{
		EntityType:   req.EntityType,
		EntityID:     req.EntityID,
		Channel:      req.Channel,
		Recipient:    req.Recipient,
		Body:         req.Body,
		Status:        models.NotificationStatusPending,
		Priority:      priority,
		RetryCount:   0,
		MaxRetries:   maxRetries,
		NextRetryAt:  sql.NullTime{}, // NULL means ready to send immediately
	}

	if req.Subject != nil {
		notification.Subject = sql.NullString{String: *req.Subject, Valid: true}
	}
	if req.TemplateID != nil {
		notification.TemplateID = sql.NullString{String: *req.TemplateID, Valid: true}
	}
	if templateDataJSON.Valid {
		notification.TemplateData = templateDataJSON
	}

	// Save to database (this is the "queue")
	err := s.repo.CreateNotification(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to queue notification: %w", err)
	}

	// Log to audit_log (REQUIRED)
	err = s.logNotificationTriggered(notification)
	if err != nil {
		// Log error but don't fail - audit logging should be resilient
		// Notification is already queued, so we continue
	}

	return &models.NotificationResult{
		NotificationID: notification.NotificationID,
		Status:         models.NotificationStatusPending,
		Success:        true,
		Message:        "Notification queued successfully",
	}, nil
}

// ProcessNotification processes a single notification
// This is called by the worker for each pending notification
func (s *NotificationService) ProcessNotification(ctx context.Context, notification *models.Notification) error {
	// Get sender for the channel
	sender, exists := s.senders[notification.Channel]
	if !exists {
		return fmt.Errorf("unsupported channel: %s", notification.Channel)
	}

	// Validate notification
	err := sender.Validate(notification)
	if err != nil {
		// Validation failed - mark as failed
		errorMsg := fmt.Sprintf("validation failed: %v", err)
		return s.handleNotificationFailure(notification, errorMsg)
	}

	// Attempt to send
	attemptNumber := notification.RetryCount + 1
	err = sender.Send(ctx, notification)
	
	// Log attempt (always log, regardless of success/failure)
	logErr := s.logNotificationAttempt(notification, attemptNumber, err)
	if logErr != nil {
		// Log error but don't fail - attempt logging should be resilient
	}

	if err != nil {
		// Sending failed - handle retry or mark as failed
		return s.handleNotificationFailure(notification, err.Error())
	}

	// Success - update status
	err = s.repo.UpdateNotificationStatus(
		notification.NotificationID,
		models.NotificationStatusSent,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	return nil
}

// handleNotificationFailure handles notification failure with retry logic
func (s *NotificationService) handleNotificationFailure(
	notification *models.Notification,
	errorMessage string,
) error {
	// Check if we should retry
	if notification.RetryCount >= notification.MaxRetries {
		// Max retries exceeded - mark as failed
		err := s.repo.UpdateNotificationStatus(
			notification.NotificationID,
			models.NotificationStatusFailed,
			&errorMessage,
		)
		if err != nil {
			return fmt.Errorf("failed to mark notification as failed: %w", err)
		}
		return fmt.Errorf("max retries exceeded: %s", errorMessage)
	}

	// Calculate next retry time with exponential backoff
	nextRetryAt := s.calculateNextRetryTime(notification.RetryCount)
	
	// Schedule retry
	err := s.repo.ScheduleRetry(
		notification.NotificationID,
		nextRetryAt,
		errorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to schedule retry: %w", err)
	}

	return fmt.Errorf("notification failed, retry scheduled: %s", errorMessage)
}

// calculateNextRetryTime calculates the next retry time using exponential backoff
// Backoff formula: delay = min(initialDelay * (multiplier ^ retryCount), maxDelay)
func (s *NotificationService) calculateNextRetryTime(retryCount int) time.Time {
	delaySeconds := s.config.InitialRetryDelay.Seconds() * math.Pow(s.config.BackoffMultiplier, float64(retryCount))
	delay := time.Duration(delaySeconds) * time.Second
	
	// Cap at max delay
	if delay > s.config.MaxRetryDelay {
		delay = s.config.MaxRetryDelay
	}

	return time.Now().Add(delay)
}

// logNotificationAttempt logs a notification attempt to notification_attempts_log
func (s *NotificationService) logNotificationAttempt(
	notification *models.Notification,
	attemptNumber int,
	sendError error,
) error {
	log := &models.NotificationLog{
		NotificationID: notification.NotificationID,
		AttemptNumber:  attemptNumber,
		Status:         models.NotificationStatusSent,
	}

	if sendError != nil {
		log.Status = models.NotificationStatusFailed
		log.ErrorMessage = sql.NullString{String: sendError.Error(), Valid: true}
	}

	// In production, you might want to include response data from the sender
	// For now, we'll leave it null

	err := s.repo.CreateNotificationAttemptLog(log)
	if err != nil {
		return fmt.Errorf("failed to log notification attempt: %w", err)
	}

	return nil
}

// logNotificationTriggered logs notification trigger to audit_log
func (s *NotificationService) logNotificationTriggered(notification *models.Notification) error {
	metadata := map[string]interface{}{
		"notification_id": notification.NotificationID,
		"channel":         notification.Channel,
		"recipient":       notification.Recipient,
		"priority":        notification.Priority,
		"max_retries":     notification.MaxRetries,
	}

	if notification.TemplateID.Valid {
		metadata["template_id"] = notification.TemplateID.String
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	auditLog := &models.AuditLog{
		EntityType:   notification.EntityType,
		EntityID:     notification.EntityID,
		Action:       "notification_triggered",
		ActionByType: models.ActorSystem,
		Metadata:     sql.NullString{String: string(metadataJSON), Valid: true},
	}

	err = s.complaintRepo.CreateAuditLog(auditLog)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetPendingNotifications retrieves pending notifications (used by worker)
func (s *NotificationService) GetPendingNotifications(limit int) ([]models.Notification, error) {
	return s.repo.GetPendingNotifications(limit)
}
