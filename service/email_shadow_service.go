package service

import (
	"context"
	"database/sql"
	"finalneta/models"
	"finalneta/notification"
	"finalneta/repository"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// PilotInboxEmail is the pilot inbox where all authority emails are sent (shadow mode)
// Exported for use in dry-run logging
const PilotInboxEmail = "aineta502@gmail.com"

// EmailShadowService implements email shadow mode for pilot: log full content, send only to pilot inbox, async.
// Authority abstraction: department_id + level (L1/L2/L3), not officer-based.
type EmailShadowService struct {
	emailLogRepo *repository.EmailLogRepository
	emailSender  *notification.EmailSender
}

// NewEmailShadowService creates a new email shadow service
func NewEmailShadowService(
	emailLogRepo *repository.EmailLogRepository,
) *EmailShadowService {
	return &EmailShadowService{
		emailLogRepo: emailLogRepo,
		emailSender:  notification.NewEmailSender(),
	}
}

// authorityDashboardPath returns the path /authority/complaints/{id}; if FRONTEND_URL is set, returns full URL (no token).
func authorityDashboardPath(complaintID int64) string {
	base := strings.TrimSuffix(os.Getenv("FRONTEND_URL"), "/")
	if base != "" {
		return fmt.Sprintf("%s/authority/complaints/%d", base, complaintID)
	}
	return fmt.Sprintf("/authority/complaints/%d", complaintID)
}

// SendAssignmentEmailAsync queues assignment email (log + send to pilot inbox). Non-blocking; never fails the caller.
func (s *EmailShadowService) SendAssignmentEmailAsync(
	complaintID int64,
	complaintNumber string,
	departmentID int64,
	departmentName string,
) {
	go s.sendAssignmentEmail(complaintID, complaintNumber, departmentID, departmentName)
}

func (s *EmailShadowService) sendAssignmentEmail(
	complaintID int64,
	complaintNumber string,
	departmentID int64,
	departmentName string,
) {
	intendedLevel := "L1"
	authorityName := departmentName
	subject := fmt.Sprintf("[AI Neta Pilot] New complaint assigned – %s", complaintNumber)
	body := fmt.Sprintf(`Pilot shadow email – authority-level notification (shadow mode).

Complaint ID: %s
Complaint number: %s
Department: %s
Authority level: %s

Open in Authority Dashboard: %s
(Login required.)

This is a pilot run. Real authority emails are disabled.`,
		fmt.Sprintf("%d", complaintID), complaintNumber, authorityName, intendedLevel, authorityDashboardPath(complaintID))

	logEntry := &models.EmailLog{
		ComplaintID:         complaintID,
		EmailType:           models.EmailLogTypeAssignment,
		IntendedAuthorityID: sql.NullInt64{Int64: departmentID, Valid: true},
		IntendedLevel:       sql.NullString{String: intendedLevel, Valid: true},
		DepartmentID:        departmentID,
		SentToEmail:         PilotInboxEmail,
		Subject:             subject,
		Body:                body,
		Status:              "pending",
	}
	if err := s.emailLogRepo.Create(logEntry); err != nil {
		log.Printf("[email_shadow] failed to log assignment email: %v", err)
		return
	}
	if sendErr := s.sendToPilotInbox(subject, body); sendErr != nil {
		_ = s.emailLogRepo.UpdateStatus(logEntry.ID, "failed", sendErr.Error())
		log.Printf("[email_shadow] send to pilot inbox failed: %v", sendErr)
	} else {
		_ = s.emailLogRepo.UpdateStatus(logEntry.ID, "sent", "")
	}
}

// SendEscalationEmailAsync queues escalation email. Non-blocking; never fails the caller.
// Authority abstraction: department_id + level (L1/L2/L3), not officer-based.
func (s *EmailShadowService) SendEscalationEmailAsync(
	complaintID int64,
	complaintNumber string,
	escalationLevel int,
	toDepartmentID int64,
	departmentName string,
	reason string,
) {
	go s.sendEscalationEmail(complaintID, complaintNumber, escalationLevel, toDepartmentID, departmentName, reason)
}

func (s *EmailShadowService) sendEscalationEmail(
	complaintID int64,
	complaintNumber string,
	escalationLevel int,
	toDepartmentID int64,
	departmentName string,
	reason string,
) {
	levelStr := fmt.Sprintf("L%d", escalationLevel)
	authorityName := departmentName
	subject := fmt.Sprintf("[AI Neta Pilot] Escalation L%d – %s", escalationLevel, complaintNumber)
	body := fmt.Sprintf(`Pilot shadow email – authority-level escalation notification (shadow mode).

Complaint ID: %s
Complaint number: %s
Escalation level: %s
Department: %s
Authority level: %s
Reason: %s

Open in Authority Dashboard: %s
(Login required.)

This is a pilot run. Real authority emails are disabled.`,
		fmt.Sprintf("%d", complaintID), complaintNumber, levelStr, authorityName, levelStr, reason, authorityDashboardPath(complaintID))

	logEntry := &models.EmailLog{
		ComplaintID:         complaintID,
		EmailType:           models.EmailLogTypeEscalation,
		IntendedAuthorityID: sql.NullInt64{Int64: toDepartmentID, Valid: true},
		IntendedLevel:       sql.NullString{String: levelStr, Valid: true},
		DepartmentID:        toDepartmentID,
		SentToEmail:         PilotInboxEmail,
		Subject:             subject,
		Body:                body,
		Status:              "pending",
	}
	if err := s.emailLogRepo.Create(logEntry); err != nil {
		log.Printf("[email_shadow] failed to log escalation email: %v", err)
		return
	}
	if sendErr := s.sendToPilotInbox(subject, body); sendErr != nil {
		_ = s.emailLogRepo.UpdateStatus(logEntry.ID, "failed", sendErr.Error())
		log.Printf("[email_shadow] send escalation failed: %v", sendErr)
	} else {
		_ = s.emailLogRepo.UpdateStatus(logEntry.ID, "sent", "")
	}
}

// SendResolutionEmailAsync queues resolution/closure email. Non-blocking; never fails the caller.
// Authority abstraction: department_id, not officer-based.
func (s *EmailShadowService) SendResolutionEmailAsync(
	complaintID int64,
	complaintNumber string,
	departmentID int64,
	departmentName string,
	newStatus string,
	reason string,
) {
	go s.sendResolutionEmail(complaintID, complaintNumber, departmentID, departmentName, newStatus, reason)
}

func (s *EmailShadowService) sendResolutionEmail(
	complaintID int64,
	complaintNumber string,
	departmentID int64,
	departmentName string,
	newStatus string,
	reason string,
) {
	authorityName := departmentName
	subject := fmt.Sprintf("[AI Neta Pilot] Complaint %s – %s", newStatus, complaintNumber)
	body := fmt.Sprintf(`Pilot shadow email – authority-level resolution notification (shadow mode).

Complaint ID: %s
Complaint number: %s
New status: %s
Department: %s
Authority: %s
Reason: %s

This is a pilot run. Real authority emails are disabled.`,
		fmt.Sprintf("%d", complaintID), complaintNumber, newStatus, authorityName, authorityName, reason)

	logEntry := &models.EmailLog{
		ComplaintID:         complaintID,
		EmailType:           models.EmailLogTypeResolution,
		IntendedAuthorityID: sql.NullInt64{Int64: departmentID, Valid: true},
		IntendedLevel:       sql.NullString{Valid: false},
		DepartmentID:        departmentID,
		SentToEmail:         PilotInboxEmail,
		Subject:             subject,
		Body:                body,
		Status:              "pending",
	}
	if err := s.emailLogRepo.Create(logEntry); err != nil {
		log.Printf("[email_shadow] failed to log resolution email: %v", err)
		return
	}
	if sendErr := s.sendToPilotInbox(subject, body); sendErr != nil {
		_ = s.emailLogRepo.UpdateStatus(logEntry.ID, "failed", sendErr.Error())
		log.Printf("[email_shadow] send resolution failed: %v", sendErr)
	} else {
		_ = s.emailLogRepo.UpdateStatus(logEntry.ID, "sent", "")
	}
}

var sendMu sync.Mutex

// sendToPilotInbox sends email to pilot inbox only. Returns error so caller can log status in email_logs.
func (s *EmailShadowService) sendToPilotInbox(subject, body string) error {
	sendMu.Lock()
	defer sendMu.Unlock()
	n := &models.Notification{
		Recipient: PilotInboxEmail,
		Subject:   sql.NullString{String: subject, Valid: true},
		Body:      body,
	}
	return s.emailSender.Send(context.Background(), n)
}
