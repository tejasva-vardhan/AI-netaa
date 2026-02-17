package handler

import (
	"finalneta/service"
	"finalneta/worker"
	"net/http"
)

// EscalationHandler handles HTTP requests for escalation operations
type EscalationHandler struct {
	escalationService *service.EscalationService
	worker            *worker.EscalationWorker
}

// NewEscalationHandler creates a new escalation handler
func NewEscalationHandler(
	escalationService *service.EscalationService,
	worker *worker.EscalationWorker,
) *EscalationHandler {
	return &EscalationHandler{
		escalationService: escalationService,
		worker:            worker,
	}
}

// ProcessEscalations handles POST /api/v1/escalations/process
// Manually triggers escalation processing (useful for testing or manual runs)
func (h *EscalationHandler) ProcessEscalations(w http.ResponseWriter, r *http.Request) {
	_ = h.worker // Worker field reserved for future use
	results, err := h.escalationService.ProcessEscalations()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"processed": len(results),
		"results":   results,
	})
}
