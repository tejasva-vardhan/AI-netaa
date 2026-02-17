package worker

import (
	"finalneta/service"
	"log"
	"time"
)

// EscalationWorker is a background worker that periodically processes escalations
type EscalationWorker struct {
	escalationService *service.EscalationService
	interval          time.Duration
	stopChan          chan struct{}
	running           bool
}

// NewEscalationWorker creates a new escalation worker
func NewEscalationWorker(
	escalationService *service.EscalationService,
	interval time.Duration,
) *EscalationWorker {
	return &EscalationWorker{
		escalationService: escalationService,
		interval:          interval,
		stopChan:         make(chan struct{}),
		running:          false,
	}
}

// Start starts the escalation worker
// The worker runs in a separate goroutine and processes escalations periodically
func (w *EscalationWorker) Start() {
	if w.running {
		log.Println("Escalation worker is already running")
		return
	}

	w.running = true
	log.Printf("Escalation worker started (interval: %v)", w.interval)

	go w.run()
}

// Stop stops the escalation worker
func (w *EscalationWorker) Stop() {
	if !w.running {
		return
	}

	log.Println("Stopping escalation worker...")
	close(w.stopChan)
	w.running = false
	log.Println("Escalation worker stopped")
}

// run is the main worker loop
func (w *EscalationWorker) run() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Process immediately on start
	w.processEscalations()

	for {
		select {
		case <-ticker.C:
			// Periodic processing
			w.processEscalations()
		case <-w.stopChan:
			// Stop signal received
			return
		}
	}
}

// processEscalations processes all escalations
// This method is idempotent - safe to call multiple times
func (w *EscalationWorker) processEscalations() {
	startTime := time.Now()
	log.Println("Starting escalation processing...")

	results, err := w.escalationService.ProcessEscalations()
	if err != nil {
		log.Printf("Error processing escalations: %v", err)
		return
	}

	duration := time.Since(startTime)
	escalatedCount := 0
	reminderCount := 0

	for _, result := range results {
		if result.Escalated {
			escalatedCount++
			log.Printf("Escalated complaint #%d: %s", result.ComplaintID, result.Reason)
		} else if result.Reason != "" {
			reminderCount++
			log.Printf("Reminder sent for complaint #%d: %s", result.ComplaintID, result.Reason)
		}
	}

	log.Printf("Escalation processing completed in %v: %d escalated, %d reminders", 
		duration, escalatedCount, reminderCount)
}
