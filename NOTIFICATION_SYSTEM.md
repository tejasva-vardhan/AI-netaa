# Notification System

## Overview

A robust, asynchronous notification system that supports multiple channels (Email, SMS, WhatsApp) with retry logic, status tracking, and complete audit trails. The system is designed to be non-blocking, resilient, and safe under high load.

## Architecture

### Components

1. **Notification Interfaces** (`notification/sender.go`)
   - `Sender` interface for all notification channels
   - `EmailSender`, `SMSSender`, `WhatsAppSender` implementations
   - Mock implementations (no real sending)

2. **Notification Repository** (`repository/notification_repository.go`)
   - Database operations for notifications
   - Status tracking
   - Retry scheduling

3. **Notification Service** (`service/notification_service.go`)
   - Queue management
   - Retry logic with exponential backoff
   - Integration with audit_log

4. **Notification Worker** (`worker/notification_worker.go`)
   - Background processing
   - Batch processing
   - Periodic execution

## Data Flow

```
1. Application triggers notification
   ↓
2. QueueNotification() called (non-blocking)
   ↓
3. Notification saved to notifications_log (status: pending)
   ↓
4. Audit log entry created
   ↓
5. Background worker picks up notification
   ↓
6. ProcessNotification() called
   ↓
7. Sender.Send() attempts to send
   ↓
8. Success → Update status to "sent"
   Failure → Schedule retry or mark as "failed"
   ↓
9. Log attempt to notification_attempts_log
```

## Database Schema

### notifications_log Table

```sql
CREATE TABLE notifications_log (
    notification_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    entity_type VARCHAR(100) NOT NULL,
    entity_id BIGINT NOT NULL,
    channel ENUM('email', 'sms', 'whatsapp') NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    subject VARCHAR(500) NULL,
    body TEXT NOT NULL,
    template_id VARCHAR(100) NULL,
    template_data JSON NULL,
    status ENUM('pending', 'sent', 'failed', 'retrying') NOT NULL DEFAULT 'pending',
    priority ENUM('low', 'normal', 'high', 'urgent') NOT NULL DEFAULT 'normal',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    next_retry_at TIMESTAMP NULL,
    sent_at TIMESTAMP NULL,
    failed_at TIMESTAMP NULL,
    error_message TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    INDEX idx_status_retry (status, next_retry_at),
    INDEX idx_entity (entity_type, entity_id),
    INDEX idx_priority_status (priority, status)
);
```

### notification_attempts_log Table

```sql
CREATE TABLE notification_attempts_log (
    log_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    notification_id BIGINT NOT NULL,
    attempt_number INT NOT NULL,
    status ENUM('pending', 'sent', 'failed', 'retrying') NOT NULL,
    error_message TEXT NULL,
    response_data JSON NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (notification_id) REFERENCES notifications_log(notification_id),
    INDEX idx_notification (notification_id, attempt_number)
);
```

## Notification Channels

### Email Channel

```go
sender := notification.NewEmailSender()
// In production, configure SMTP or email service API
```

**Mock Implementation**: Currently returns success immediately
**Production**: Integrate with SendGrid, AWS SES, SMTP, etc.

### SMS Channel

```go
sender := notification.NewSMSSender()
// In production, configure SMS gateway
```

**Mock Implementation**: Currently returns success immediately
**Production**: Integrate with Twilio, AWS SNS, etc.

### WhatsApp Channel

```go
sender := notification.NewWhatsAppSender()
// In production, configure WhatsApp Business API
```

**Mock Implementation**: Currently returns success immediately
**Production**: Integrate with WhatsApp Business API

## Usage

### Queue a Notification

```go
req := &models.NotificationRequest{
    EntityType: "complaint",
    EntityID:   123,
    Channel:    models.ChannelEmail,
    Recipient:  "user@example.com",
    Subject:    stringPtr("Complaint Status Update"),
    Body:       "Your complaint has been verified.",
    Priority:   models.NotificationPriorityNormal,
    MaxRetries: intPtr(3),
}

result, err := notificationService.QueueNotification(req)
// Returns immediately - notification queued for async processing
```

### Template Support

```go
req := &models.NotificationRequest{
    EntityType: "complaint",
    EntityID:   123,
    Channel:    models.ChannelEmail,
    Recipient:  "user@example.com",
    TemplateID: stringPtr("complaint_verified"),
    TemplateData: map[string]interface{}{
        "complaint_number": "COMP-20260212-abc123",
        "status": "verified",
        "user_name": "John Doe",
    },
}
```

**Note**: Template rendering is handled by the sender implementation. The system stores template_id and template_data but doesn't render templates itself.

## Retry Strategy

### Exponential Backoff

Retry delays increase exponentially:

```
Retry 1: 1 minute
Retry 2: 2 minutes
Retry 3: 4 minutes
Retry 4: 8 minutes
...
Max delay: 30 minutes (configurable)
```

### Configuration

```go
config := &models.NotificationConfig{
    DefaultMaxRetries: 3,
    InitialRetryDelay: 1 * time.Minute,
    MaxRetryDelay:     30 * time.Minute,
    BackoffMultiplier: 2.0,
}
```

### Retry Logic

1. **First Failure**: Schedule retry with initial delay
2. **Subsequent Failures**: Increase delay exponentially
3. **Max Retries Exceeded**: Mark as "failed", stop retrying
4. **Status Tracking**: Status changes to "retrying" during retries

## Status Tracking

### Status Flow

```
pending → sent (success)
pending → retrying → sent (success after retry)
pending → retrying → failed (max retries exceeded)
```

### Status Values

- **pending**: Initial state, ready to send
- **retrying**: Failed, scheduled for retry
- **sent**: Successfully sent
- **failed**: Max retries exceeded

## Worker Configuration

### Default Configuration

```go
worker := worker.NewNotificationWorker(
    notificationService,
    30*time.Second, // Process every 30 seconds
)
worker.Start()
```

### Batch Processing

Worker processes notifications in batches:
- Default batch size: 100 notifications
- Processes highest priority first
- Processes oldest notifications first within same priority

## Failure Handling

### Non-Blocking Design

- **QueueNotification()**: Returns immediately, doesn't wait for send
- **ProcessNotification()**: Errors don't block other notifications
- **Audit Logging**: Failures don't prevent notification queuing

### Resilient Components

1. **Notification Queuing**: Always succeeds (writes to DB)
2. **Audit Logging**: Failures logged but don't block
3. **Attempt Logging**: Failures logged but don't block
4. **Status Updates**: Failures logged but don't block

### Error Handling Strategy

```go
// Queue notification (always succeeds)
result, err := notificationService.QueueNotification(req)
if err != nil {
    // Only fails if DB write fails
    // In production, consider retry or fallback
}

// Process notification (errors handled internally)
// Worker continues processing other notifications even if one fails
```

## Audit Logging

### Notification Triggered

When a notification is queued:

```json
{
  "entity_type": "complaint",
  "entity_id": 123,
  "action": "notification_triggered",
  "action_by_type": "system",
  "metadata": {
    "notification_id": 1,
    "channel": "email",
    "recipient": "user@example.com",
    "priority": "normal",
    "max_retries": 3,
    "template_id": "complaint_verified"
  }
}
```

### Attempt Logging

Every send attempt is logged:

```json
{
  "notification_id": 1,
  "attempt_number": 1,
  "status": "sent",
  "error_message": null,
  "response_data": null
}
```

## Integration Points

### Complaint Status Changes

```go
// After status update
if statusChanged {
    // Queue notification (non-blocking)
    notificationService.QueueNotification(&models.NotificationRequest{
        EntityType: "complaint",
        EntityID:   complaintID,
        Channel:    models.ChannelEmail,
        Recipient:  userEmail,
        TemplateID: stringPtr("status_update"),
        TemplateData: map[string]interface{}{
            "complaint_number": complaintNumber,
            "old_status": oldStatus,
            "new_status": newStatus,
        },
    })
    // Continue with complaint processing - notification is async
}
```

### Escalation Notifications

```go
// After escalation
notificationService.QueueNotification(&models.NotificationRequest{
    EntityType: "complaint",
    EntityID:   complaintID,
    Channel:    models.ChannelEmail,
    Recipient:  officerEmail,
    Priority:   models.NotificationPriorityHigh,
    TemplateID: stringPtr("escalation"),
})
```

## Performance Considerations

### High Load Safety

1. **Non-Blocking**: Queue operations don't block request processing
2. **Batch Processing**: Worker processes multiple notifications efficiently
3. **Database Indexing**: Proper indexes on status, priority, next_retry_at
4. **Connection Pooling**: Database connections reused
5. **Rate Limiting**: Can be added at sender level

### Scalability

- **Horizontal Scaling**: Multiple workers can run simultaneously
- **Database Queue**: Uses database as queue (no external queue needed)
- **Idempotent Processing**: Safe to process same notification multiple times
- **Batch Size**: Configurable batch size for optimal throughput

## Monitoring

### Key Metrics

1. **Pending Notifications**: Count of pending/retrying notifications
2. **Success Rate**: Percentage of successfully sent notifications
3. **Retry Rate**: Percentage of notifications requiring retries
4. **Average Retry Count**: Average number of retries before success
5. **Processing Time**: Time to process batch of notifications

### Logging

Worker logs:
- Processing start/end times
- Number of notifications processed
- Success/failure counts
- Retry counts

## Testing

### Unit Testing

```go
// Test notification queuing
req := &models.NotificationRequest{...}
result, err := service.QueueNotification(req)
assert.NoError(t, err)
assert.Equal(t, models.NotificationStatusPending, result.Status)

// Test notification processing
err := service.ProcessNotification(ctx, notification)
assert.NoError(t, err)
```

### Integration Testing

```go
// Test worker processing
worker.Start()
time.Sleep(2 * time.Second) // Wait for processing
worker.Stop()

// Verify notifications processed
notifications := getNotificationsByStatus("sent")
assert.Greater(t, len(notifications), 0)
```

## Future Enhancements

1. **Template Engine**: Add template rendering (Mustache, Go templates)
2. **Rate Limiting**: Per-channel rate limiting
3. **Dead Letter Queue**: Store permanently failed notifications
4. **Webhooks**: Trigger webhooks on notification events
5. **Analytics**: Track delivery rates, open rates (for email)
6. **Multi-language**: Support for multiple languages
7. **Scheduled Notifications**: Send notifications at specific times
8. **Notification Preferences**: User preferences for channels

## Best Practices

1. **Always Use Templates**: Don't hardcode email/SMS content
2. **Set Appropriate Priorities**: Use urgent for critical notifications
3. **Monitor Retry Rates**: High retry rates indicate issues
4. **Test Senders**: Test each channel before production
5. **Handle Failures Gracefully**: Don't block business logic on notification failures
6. **Log Everything**: All attempts logged for debugging
7. **Monitor Queue Size**: Alert if queue grows too large
