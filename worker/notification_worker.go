package worker

import (
	"context"
	"finalneta/models"
	"finalneta/service"
	"log"
	"time"
)

// NotificationWorker is a background worker that processes notifications asynchronously
type NotificationWorker struct {
	notificationService *service.NotificationService
	interval            time.Duration
	stopChan            chan struct{}
	running             bool
}

// NewNotificationWorker creates a new notification worker
func NewNotificationWorker(
	notificationService *service.NotificationService,
	interval time.Duration,
) *NotificationWorker {
	return &NotificationWorker{
		notificationService: notificationService,
		interval:            interval,
		stopChan:            make(chan struct{}),
		running:             false,
	}
}

// Start starts the notification worker
// The worker runs in a separate goroutine and processes notifications periodically
func (w *NotificationWorker) Start() {
	if w.running {
		log.Println("Notification worker is already running")
		return
	}

	w.running = true
	log.Printf("Notification worker started (interval: %v)", w.interval)

	go w.run()
}

// Stop stops the notification worker
func (w *NotificationWorker) Stop() {
	if !w.running {
		return
	}

	log.Println("Stopping notification worker...")
	close(w.stopChan)
	w.running = false
	log.Println("Notification worker stopped")
}

// run is the main worker loop
func (w *NotificationWorker) run() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Process immediately on start
	w.processNotifications()

	for {
		select {
		case <-ticker.C:
			// Periodic processing
			w.processNotifications()
		case <-w.stopChan:
			// Stop signal received
			return
		}
	}
}

// processNotifications processes pending notifications
// This method is safe to call multiple times (idempotent)
func (w *NotificationWorker) processNotifications() {
	startTime := time.Now()
	log.Println("Starting notification processing...")

	// Get pending notifications (batch processing)
	notifications, err := w.notificationService.GetPendingNotifications(100) // Batch size
	if err != nil {
		log.Printf("Error getting pending notifications: %v", err)
		return
	}

	if len(notifications) == 0 {
		log.Println("No pending notifications to process")
		return
	}

	log.Printf("Processing %d notifications...", len(notifications))

	ctx := context.Background()
	successCount := 0
	failedCount := 0
	retryCount := 0

	// Process each notification
	for _, notification := range notifications {
		err := w.notificationService.ProcessNotification(ctx, &notification)
		if err != nil {
			// Check if it's a retry (not a final failure)
			if notification.Status == models.NotificationStatusRetrying {
				retryCount++
				log.Printf("Notification #%d scheduled for retry: %v", notification.NotificationID, err)
			} else {
				failedCount++
				log.Printf("Notification #%d failed: %v", notification.NotificationID, err)
			}
		} else {
			successCount++
			log.Printf("Notification #%d sent successfully", notification.NotificationID)
		}
	}

	duration := time.Since(startTime)
	log.Printf("Notification processing completed in %v: %d sent, %d failed, %d retries",
		duration, successCount, failedCount, retryCount)
}
