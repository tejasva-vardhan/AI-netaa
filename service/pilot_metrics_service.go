package service

import (
	"finalneta/models"
	"finalneta/repository"
	"log"
	"time"
)

// PilotMetricsService handles pilot metrics event emission
type PilotMetricsService struct {
	metricsRepo *repository.PilotMetricsRepository
}

// NewPilotMetricsService creates a new pilot metrics service
func NewPilotMetricsService(metricsRepo *repository.PilotMetricsRepository) *PilotMetricsService {
	return &PilotMetricsService{
		metricsRepo: metricsRepo,
	}
}

// EmitComplaintCreated emits a complaint_created event
func (s *PilotMetricsService) EmitComplaintCreated(complaintID, userID int64, metadata map[string]interface{}) {
	if s.metricsRepo == nil {
		return // Service not initialized
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["timestamp"] = time.Now().Unix()

	err := s.metricsRepo.CreateEventWithMetadata(
		models.EventComplaintCreated,
		&complaintID,
		&userID,
		metadata,
	)
	if err != nil {
		log.Printf("[METRICS] Failed to emit complaint_created event: %v", err)
	}
}

// EmitFirstAuthorityAction emits a first_authority_action event
// Calculates time_to_first_authority_action from complaint creation
func (s *PilotMetricsService) EmitFirstAuthorityAction(complaintID, userID int64, complaintCreatedAt time.Time, metadata map[string]interface{}) {
	if s.metricsRepo == nil {
		return
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	// Calculate time to first authority action
	timeToAction := time.Since(complaintCreatedAt)
	metadata["time_to_first_action_seconds"] = int64(timeToAction.Seconds())
	metadata["time_to_first_action_hours"] = timeToAction.Hours()
	metadata["timestamp"] = time.Now().Unix()

	err := s.metricsRepo.CreateEventWithMetadata(
		models.EventFirstAuthorityAction,
		&complaintID,
		&userID,
		metadata,
	)
	if err != nil {
		log.Printf("[METRICS] Failed to emit first_authority_action event: %v", err)
	}
}

// EmitEscalationTriggered emits an escalation_triggered event
func (s *PilotMetricsService) EmitEscalationTriggered(complaintID, userID int64, escalationLevel int, metadata map[string]interface{}) {
	if s.metricsRepo == nil {
		return
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["escalation_level"] = escalationLevel
	metadata["timestamp"] = time.Now().Unix()

	err := s.metricsRepo.CreateEventWithMetadata(
		models.EventEscalationTriggered,
		&complaintID,
		&userID,
		metadata,
	)
	if err != nil {
		log.Printf("[METRICS] Failed to emit escalation_triggered event: %v", err)
	}
}

// EmitComplaintResolved emits a complaint_resolved event
// Calculates time to resolution from complaint creation
func (s *PilotMetricsService) EmitComplaintResolved(complaintID, userID int64, complaintCreatedAt time.Time, status string, metadata map[string]interface{}) {
	if s.metricsRepo == nil {
		return
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	// Calculate time to resolution
	timeToResolution := time.Since(complaintCreatedAt)
	metadata["time_to_resolution_seconds"] = int64(timeToResolution.Seconds())
	metadata["time_to_resolution_hours"] = timeToResolution.Hours()
	metadata["status"] = status
	metadata["timestamp"] = time.Now().Unix()

	err := s.metricsRepo.CreateEventWithMetadata(
		models.EventComplaintResolved,
		&complaintID,
		&userID,
		metadata,
	)
	if err != nil {
		log.Printf("[METRICS] Failed to emit complaint_resolved event: %v", err)
	}
}

// EmitChatAbandoned emits a chat_abandoned event
func (s *PilotMetricsService) EmitChatAbandoned(userID int64, metadata map[string]interface{}) {
	if s.metricsRepo == nil {
		return
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["timestamp"] = time.Now().Unix()

	err := s.metricsRepo.CreateEventWithMetadata(
		models.EventChatAbandoned,
		nil, // No complaint_id for abandoned chats
		&userID,
		metadata,
	)
	if err != nil {
		log.Printf("[METRICS] Failed to emit chat_abandoned event: %v", err)
	}
}
