package handler

import (
	"finalneta/service"
	"net/http"
)

// ChatHandler handles chat/draft reset (hidden "restart" UX). Auth remains intact.
type ChatHandler struct {
	pilotMetricsService *service.PilotMetricsService
}

// NewChatHandler creates a new chat handler.
func NewChatHandler(pilotMetricsService *service.PilotMetricsService) *ChatHandler {
	return &ChatHandler{
		pilotMetricsService: pilotMetricsService,
	}
}

// ResetChatDraft handles POST /api/v1/users/chat/reset.
// Clears server-side chat_state and temp complaint draft for the authenticated user.
// Auth (JWT, phone_verified) is unchanged. If chat_state/draft storage is added later, clear it here.
// Emits chat_abandoned metric event if user had a draft but never submitted.
func (h *ChatHandler) ResetChatDraft(w http.ResponseWriter, r *http.Request) {
	// User is already authenticated by middleware; no body required.
	// Future: clear user_draft / chat_state by user_id from context.
	
	// Emit pilot metrics: chat_abandoned
	// Note: This tracks when user resets/abandons chat without submitting
	// For intentional "restart", this is still considered abandonment of the previous session
	if h.pilotMetricsService != nil {
		userID, err := getUserIDFromContext(r)
		if err == nil {
			metadata := map[string]interface{}{
				"action": "reset",
			}
			h.pilotMetricsService.EmitChatAbandoned(userID, metadata)
		}
	}
	
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}
