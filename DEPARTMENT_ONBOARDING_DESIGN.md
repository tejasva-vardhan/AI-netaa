# Department Onboarding Mechanism Design

## Overview

A safe, neutral, and opt-in based onboarding system for departments in the public accountability platform pilot. The system respects department autonomy, maintains neutrality, and ensures proper verification while allowing the platform to function even if departments don't respond.

## Design Principles

1. **Opt-In Only**: Departments must explicitly acknowledge/opt-in
2. **Neutral Communication**: Informational, non-accusatory tone
3. **No Forced Action**: Platform works even if departments don't respond
4. **Verification Required**: Email verification before active participation
5. **Complete Audit Trail**: All communications logged

## Onboarding Flow

### Phase 1: Initial Contact

```
[System Admin adds department contact email]
  ↓
[System generates verification token]
  ↓
[System sends introductory email]
  ↓
[Email logged in audit]
  ↓
[Department status: "pending_verification"]
  ↓
WAIT FOR RESPONSE (7 days)
```

### Phase 2: Email Verification

```
[Department receives email]
  ↓
[Department clicks verification link]
  ↓
[System verifies token]
  ↓
[Department status: "email_verified"]
  ↓
[System sends acknowledgement email]
  ↓
[Email logged in audit]
```

### Phase 3: Opt-In Acknowledgement

```
[Department views onboarding page]
  ↓
[Department reads information]
  ↓
[Department acknowledges/opts-in]
  ↓
[Department status: "acknowledged"]
  ↓
[System sends welcome email]
  ↓
[Email logged in audit]
  ↓
[Department becomes active participant]
```

### Phase 4: Fallback Handling

```
[If no response after 7 days]
  ↓
[System sends reminder email]
  ↓
[Email logged in audit]
  ↓
[If no response after 14 days]
  ↓
[System marks as "non_responsive"]
  ↓
[Platform continues to function]
  ↓
[Complaints still routed to department]
  ↓
[Notifications sent but no active participation]
```

## Data Model

### 1. department_onboarding

**Purpose**: Track department onboarding status and communications

```sql
CREATE TABLE department_onboarding (
    onboarding_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    department_id BIGINT NOT NULL,
    FOREIGN KEY (department_id) REFERENCES district_departments(dept_id),
    
    -- Contact information
    contact_email VARCHAR(255) NOT NULL,
    contact_name VARCHAR(255) NULL, -- Optional, not public
    contact_designation VARCHAR(255) NULL, -- e.g., "Department Head"
    
    -- Verification
    verification_token VARCHAR(64) NOT NULL UNIQUE,
    email_verified_at TIMESTAMP NULL,
    
    -- Onboarding status
    status ENUM(
        'pending_verification',
        'email_verified',
        'acknowledged',
        'active',
        'non_responsive',
        'opted_out'
    ) NOT NULL DEFAULT 'pending_verification',
    
    -- Opt-in tracking
    acknowledged_at TIMESTAMP NULL,
    acknowledged_by_email VARCHAR(255) NULL,
    acknowledgement_ip VARCHAR(45) NULL,
    
    -- Communication tracking
    intro_email_sent_at TIMESTAMP NULL,
    reminder_email_sent_at TIMESTAMP NULL,
    reminder_count INT NOT NULL DEFAULT 0,
    last_contact_at TIMESTAMP NULL,
    
    -- Metadata
    notes TEXT NULL, -- Internal notes, not public
    created_by_user_id BIGINT NULL, -- System admin who added
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    
    UNIQUE KEY uk_dept_email (department_id, contact_email),
    INDEX idx_status (status),
    INDEX idx_verification_token (verification_token)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 2. department_onboarding_communications

**Purpose**: Log all onboarding-related communications

```sql
CREATE TABLE department_onboarding_communications (
    communication_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    onboarding_id BIGINT NOT NULL,
    FOREIGN KEY (onboarding_id) REFERENCES department_onboarding(onboarding_id),
    
    -- Communication details
    communication_type ENUM(
        'intro_email',
        'verification_email',
        'acknowledgement_email',
        'reminder_email',
        'welcome_email',
        'opt_out_email'
    ) NOT NULL,
    
    -- Email details
    email_subject VARCHAR(500) NOT NULL,
    email_body TEXT NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,
    
    -- Delivery status
    sent_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    delivery_status ENUM('sent', 'delivered', 'bounced', 'failed') NOT NULL DEFAULT 'sent',
    delivery_error TEXT NULL,
    
    -- Response tracking
    opened_at TIMESTAMP NULL,
    clicked_at TIMESTAMP NULL,
    clicked_link VARCHAR(500) NULL,
    
    -- Metadata
    ip_address VARCHAR(45) NULL,
    user_agent VARCHAR(500) NULL,
    
    INDEX idx_onboarding (onboarding_id),
    INDEX idx_type (communication_type),
    INDEX idx_sent_at (sent_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3. department_onboarding_acknowledgements

**Purpose**: Track explicit acknowledgements/opt-ins

```sql
CREATE TABLE department_onboarding_acknowledgements (
    acknowledgement_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    onboarding_id BIGINT NOT NULL,
    FOREIGN KEY (onboarding_id) REFERENCES department_onboarding(onboarding_id),
    
    -- Acknowledgement details
    acknowledged_by_email VARCHAR(255) NOT NULL,
    acknowledgement_type ENUM('opt_in', 'acknowledge', 'opt_out') NOT NULL,
    
    -- What was acknowledged
    acknowledged_items JSON NOT NULL, -- ['email_verification', 'platform_terms', 'data_sharing']
    
    -- Technical details
    ip_address VARCHAR(45) NULL,
    user_agent VARCHAR(500) NULL,
    verification_token VARCHAR(64) NULL,
    
    -- Timestamp
    acknowledged_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_onboarding (onboarding_id),
    INDEX idx_email (acknowledged_by_email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## Onboarding Flow Details

### Step 1: Admin Adds Department Contact

**Action**: System admin manually adds department contact email

**Data Entry**:
```sql
INSERT INTO department_onboarding (
    department_id,
    contact_email,
    contact_name, -- Optional, internal only
    contact_designation, -- e.g., "Department Head"
    verification_token,
    status,
    created_by_user_id
) VALUES (
    1, -- PWD department
    'pwd@shivpuri.mp.gov.in',
    NULL, -- Not required
    'Department Head',
    'verification_token_here',
    'pending_verification',
    1 -- Admin user ID
);
```

**No Public Exposure**: Contact name is internal only, never displayed publicly

---

### Step 2: System Sends Introductory Email

**Trigger**: Immediately after admin adds contact

**Email Content** (see Sample Email section below)

**Logging**:
```sql
INSERT INTO department_onboarding_communications (
    onboarding_id,
    communication_type,
    email_subject,
    email_body,
    recipient_email,
    sent_at
) VALUES (
    1,
    'intro_email',
    'Introduction: Public Accountability Platform - Shivpuri District',
    '[Email body]',
    'pwd@shivpuri.mp.gov.in',
    NOW()
);
```

**Update Status**:
```sql
UPDATE department_onboarding
SET intro_email_sent_at = NOW(),
    last_contact_at = NOW()
WHERE onboarding_id = 1;
```

---

### Step 3: Email Verification

**User Action**: Department clicks verification link in email

**Verification Link Format**:
```
https://platform.example.com/onboarding/verify?token={verification_token}&email={contact_email}
```

**Verification Process**:
1. System validates token
2. Checks token hasn't expired (7 days)
3. Updates status to "email_verified"
4. Sends acknowledgement email
5. Logs verification action

**SQL Update**:
```sql
UPDATE department_onboarding
SET email_verified_at = NOW(),
    status = 'email_verified',
    last_contact_at = NOW()
WHERE verification_token = ?
  AND contact_email = ?
  AND status = 'pending_verification';
```

**Log Communication**:
```sql
INSERT INTO department_onboarding_communications (
    onboarding_id,
    communication_type,
    email_subject,
    email_body,
    recipient_email,
    clicked_at,
    clicked_link
) VALUES (
    1,
    'verification_email',
    'Email Verified - Public Accountability Platform',
    '[Acknowledgement email body]',
    'pwd@shivpuri.mp.gov.in',
    NOW(),
    '/onboarding/verify?token=...'
);
```

---

### Step 4: Opt-In Acknowledgement

**User Action**: Department views onboarding page and acknowledges

**Onboarding Page Content**:
- Platform overview (neutral, informational)
- How complaints are routed
- What participation means
- Data sharing information
- Opt-in checkbox

**Acknowledgement Process**:
1. Department views onboarding page
2. Reads information
3. Checks acknowledgement checkbox
4. Submits form
5. System records acknowledgement
6. Updates status to "acknowledged"
7. Sends welcome email

**SQL Insert**:
```sql
INSERT INTO department_onboarding_acknowledgements (
    onboarding_id,
    acknowledged_by_email,
    acknowledgement_type,
    acknowledged_items,
    ip_address,
    user_agent,
    verification_token
) VALUES (
    1,
    'pwd@shivpuri.mp.gov.in',
    'opt_in',
    '["email_verification", "platform_terms", "data_sharing"]',
    '192.168.1.1',
    'Mozilla/5.0...',
    'verification_token'
);

UPDATE department_onboarding
SET status = 'acknowledged',
    acknowledged_at = NOW(),
    acknowledged_by_email = 'pwd@shivpuri.mp.gov.in',
    acknowledgement_ip = '192.168.1.1',
    last_contact_at = NOW()
WHERE onboarding_id = 1;
```

---

### Step 5: Active Participation

**Status Change**: "acknowledged" → "active"

**Trigger**: After acknowledgement, system marks as active

**SQL Update**:
```sql
UPDATE department_onboarding
SET status = 'active'
WHERE onboarding_id = 1
  AND status = 'acknowledged';
```

**Welcome Email**: Sent when status becomes "active"

---

## Sample Email Content

### Email 1: Introductory Email

**Subject**: Introduction: Public Accountability Platform - Shivpuri District

**Body** (Hindi + English):

```
Namaste / Greetings,

We are writing to introduce a new public accountability platform that has been 
launched in Shivpuri district, Madhya Pradesh.

**About the Platform:**
This platform enables citizens to file complaints about public issues such as 
infrastructure problems, sanitation concerns, service delivery issues, and 
other matters. All complaints are tracked transparently, and citizens can view 
the status of their complaints.

**Your Department's Role:**
Based on the district configuration, complaints related to [Department Name] 
will be routed to your department. The platform facilitates transparent 
communication between citizens and government departments.

**What This Means:**
- Citizens can file complaints related to your department's jurisdiction
- Complaints will be assigned to your department based on category
- You will receive notifications about new complaints
- Citizens can track complaint status through the platform

**Next Steps:**
To verify this email address and learn more about participation, please click 
the verification link below:

[Verification Link]

This link will expire in 7 days.

**Important Notes:**
- Participation is voluntary and based on acknowledgement
- The platform will function regardless of department response
- All communications are logged for transparency
- No individual officer names are displayed publicly

**Questions or Concerns:**
If you have questions or would like more information, please reply to this 
email or contact [Admin Contact].

Thank you for your attention.

Best regards,
Public Accountability Platform Team
Shivpuri District, Madhya Pradesh

---
यह ईमेल Shivpuri जिले में नए public accountability platform के बारे में है।
कृपया verification link पर click करें।
```

**Tone**: Neutral, informational, respectful
**No Accusations**: No mention of problems or failures
**No Pressure**: Clear that participation is voluntary

---

### Email 2: Verification Acknowledgement

**Subject**: Email Verified - Public Accountability Platform

**Body**:

```
Thank you for verifying your email address.

Your email (pwd@shivpuri.mp.gov.in) has been verified for the Public 
Accountability Platform.

**What's Next:**
To complete the onboarding process and learn more about how the platform 
works, please visit:

[Onboarding Page Link]

On this page, you can:
- Learn about the platform's features
- Understand how complaints are routed
- Review data sharing information
- Acknowledge participation (optional)

**Important:**
- The platform will continue to function even if you don't complete onboarding
- Complaints will still be routed to your department
- You can opt-out at any time

If you have questions, please reply to this email.

Best regards,
Public Accountability Platform Team
```

---

### Email 3: Reminder Email (After 7 Days)

**Subject**: Reminder: Public Accountability Platform - Email Verification

**Body**:

```
This is a reminder about the Public Accountability Platform introduction email 
sent 7 days ago.

We understand you may be busy. If you'd like to verify your email and learn 
more about the platform, please use the link below:

[Verification Link]

**Note:** This is optional. The platform will continue to function regardless 
of your response. Complaints will still be routed to your department as 
configured.

If you prefer not to participate, no action is needed.

Thank you for your time.

Best regards,
Public Accountability Platform Team
```

**Tone**: Gentle reminder, no pressure, clear that action is optional

---

### Email 4: Welcome Email (After Acknowledgement)

**Subject**: Welcome to Public Accountability Platform

**Body**:

```
Thank you for acknowledging participation in the Public Accountability Platform.

**What This Means:**
- You will receive email notifications about complaints assigned to your 
  department
- You can view complaint details through the platform
- Citizens can track complaint status
- All interactions are logged for transparency

**Access Information:**
- Department Dashboard: [Dashboard Link]
- Login credentials will be sent separately (if applicable)
- Support: [Support Email]

**Resources:**
- User Guide: [Guide Link]
- FAQ: [FAQ Link]
- Contact Support: [Support Email]

**Important:**
- You can opt-out at any time
- All communications are logged
- No individual officer names are displayed publicly

Welcome aboard!

Best regards,
Public Accountability Platform Team
```

---

## Fallback Handling

### Scenario 1: Department Doesn't Verify Email

**Timeline**:
- Day 0: Intro email sent
- Day 7: Reminder email sent
- Day 14: Status marked as "non_responsive"

**System Behavior**:
```sql
-- After 14 days, update status
UPDATE department_onboarding
SET status = 'non_responsive',
    reminder_count = 2
WHERE onboarding_id = 1
  AND status IN ('pending_verification', 'email_verified')
  AND DATEDIFF(NOW(), intro_email_sent_at) >= 14;
```

**Platform Behavior**:
- ✅ Complaints still routed to department (based on category mapping)
- ✅ Notifications sent to department email
- ✅ Platform functions normally
- ✅ No active participation (no dashboard access, no login)
- ✅ Department can verify/acknowledge later

**No Negative Impact**: Platform continues to work, complaints are still routed

---

### Scenario 2: Department Verifies Email But Doesn't Acknowledge

**Timeline**:
- Day 0: Intro email sent
- Day 2: Email verified
- Day 9: Reminder sent (for acknowledgement)
- Day 16: Status remains "email_verified"

**System Behavior**:
- Status stays "email_verified"
- Notifications sent to verified email
- No dashboard access
- Can acknowledge later

**Platform Behavior**:
- ✅ Complaints routed normally
- ✅ Email notifications sent
- ✅ No active participation features
- ✅ Can complete acknowledgement anytime

---

### Scenario 3: Department Opts Out

**User Action**: Department explicitly opts out

**Process**:
1. Department clicks opt-out link
2. System records opt-out
3. Status updated to "opted_out"
4. Opt-out email sent
5. Logged in audit

**SQL Update**:
```sql
INSERT INTO department_onboarding_acknowledgements (
    onboarding_id,
    acknowledged_by_email,
    acknowledgement_type,
    acknowledged_items
) VALUES (
    1,
    'pwd@shivpuri.mp.gov.in',
    'opt_out',
    '["opt_out"]'
);

UPDATE department_onboarding
SET status = 'opted_out',
    last_contact_at = NOW()
WHERE onboarding_id = 1;
```

**Platform Behavior**:
- ✅ Complaints still routed (routing is automatic)
- ❌ No email notifications sent
- ❌ No dashboard access
- ✅ Can opt back in later

**Opt-Out Email**:
```
Subject: Opt-Out Confirmed - Public Accountability Platform

Thank you for your response.

You have opted out of active participation in the Public Accountability Platform.

**What This Means:**
- Complaints will still be routed to your department based on category
- You will not receive email notifications
- You will not have dashboard access
- Citizens can still file complaints related to your department

**Re-joining:**
If you change your mind, you can opt back in at any time by replying to this 
email or visiting [Onboarding Page].

Thank you for your time.

Best regards,
Public Accountability Platform Team
```

---

## Audit Logging

### All Communications Logged

Every email sent is logged in `department_onboarding_communications`:

```sql
-- Log intro email
INSERT INTO department_onboarding_communications (
    onboarding_id, communication_type, email_subject,
    email_body, recipient_email, sent_at, delivery_status
) VALUES (...);

-- Log email open (if tracking pixel used)
UPDATE department_onboarding_communications
SET opened_at = NOW()
WHERE communication_id = ?;

-- Log link click
UPDATE department_onboarding_communications
SET clicked_at = NOW(),
    clicked_link = ?
WHERE communication_id = ?;
```

### All Actions Logged in Audit Log

```sql
-- Log admin action
INSERT INTO audit_log (
    entity_type, entity_id, action,
    action_by_type, action_by_user_id,
    new_values, metadata
) VALUES (
    'department_onboarding',
    1,
    'contact_added',
    'admin',
    1,
    '{"contact_email": "pwd@shivpuri.mp.gov.in"}',
    '{"department_id": 1}'
);

-- Log verification
INSERT INTO audit_log (
    entity_type, entity_id, action,
    action_by_type,
    new_values, metadata
) VALUES (
    'department_onboarding',
    1,
    'email_verified',
    'system',
    '{"status": "email_verified"}',
    '{"ip_address": "192.168.1.1"}'
);

-- Log acknowledgement
INSERT INTO audit_log (
    entity_type, entity_id, action,
    action_by_type,
    new_values, metadata
) VALUES (
    'department_onboarding',
    1,
    'acknowledged',
    'system',
    '{"status": "acknowledged"}',
    '{"acknowledgement_type": "opt_in"}'
);
```

## Status Flow Diagram

```
pending_verification
    ↓ (email verified)
email_verified
    ↓ (acknowledged)
acknowledged
    ↓ (system activation)
active
    ↓ (opt-out)
opted_out

pending_verification
    ↓ (no response after 14 days)
non_responsive

email_verified
    ↓ (no acknowledgement after 14 days)
email_verified (stays, but no active participation)
```

## Query Examples

### Get Department Onboarding Status

```sql
SELECT 
    d.department_name,
    do.contact_email,
    do.status,
    do.email_verified_at,
    do.acknowledged_at,
    do.last_contact_at,
    do.reminder_count
FROM department_onboarding do
JOIN district_departments d ON do.department_id = d.dept_id
WHERE d.district_config_id = 1
ORDER BY d.department_name;
```

### Get Pending Verifications

```sql
SELECT *
FROM department_onboarding
WHERE status = 'pending_verification'
  AND DATEDIFF(NOW(), intro_email_sent_at) >= 7
  AND reminder_count < 2;
```

### Get Communication History

```sql
SELECT 
    doc.communication_type,
    doc.email_subject,
    doc.sent_at,
    doc.delivery_status,
    doc.opened_at,
    doc.clicked_at
FROM department_onboarding_communications doc
WHERE doc.onboarding_id = ?
ORDER BY doc.sent_at DESC;
```

## Security Considerations

### 1. Token Security

- **Token Generation**: Cryptographically secure random token (64 characters)
- **Token Expiry**: 7 days for verification links
- **Single Use**: Tokens invalidated after use
- **Rate Limiting**: Max 3 verification attempts per email per day

### 2. Email Verification

- **Double Opt-In**: Email verification required before acknowledgement
- **Domain Validation**: Verify email domain matches department domain (optional)
- **Bounce Handling**: Track email bounces, mark as invalid if needed

### 3. Privacy

- **No Public Names**: Contact names never displayed publicly
- **Internal Only**: Contact names stored but only visible to admins
- **Audit Trail**: All actions logged but sensitive data protected

## Integration with Existing System

### Complaint Routing (Works Regardless of Onboarding Status)

```sql
-- When complaint is created
SELECT 
    cdm.primary_department_id,
    d.department_name,
    do.status as onboarding_status,
    do.contact_email
FROM category_department_mapping cdm
JOIN district_departments d ON cdm.primary_department_id = d.dept_id
LEFT JOIN department_onboarding do ON d.dept_id = do.department_id
WHERE cdm.category = ?
  AND cdm.district_config_id = ?
  AND cdm.is_active = TRUE;
```

**Routing Logic**:
- Route complaint to department (regardless of onboarding status)
- If onboarding status = "active": Send notification + dashboard access
- If onboarding status = "non_responsive": Send email notification only
- If onboarding status = "opted_out": Route but don't send notifications

### Notification Sending

```sql
-- Check if department should receive notifications
SELECT 
    do.contact_email,
    do.status
FROM department_onboarding do
WHERE do.department_id = ?
  AND do.status IN ('active', 'email_verified', 'acknowledged');
```

**Notification Logic**:
- Status = "active": Send all notifications
- Status = "email_verified" or "acknowledged": Send email notifications
- Status = "opted_out" or "non_responsive": Don't send notifications
- Status = "pending_verification": Don't send notifications (not verified)

## Summary

This onboarding mechanism provides:

1. **Safe & Neutral**: No accusations, informational tone, voluntary participation
2. **Opt-In Based**: Departments explicitly acknowledge/opt-in
3. **No Forced Action**: Platform works even if departments don't respond
4. **Complete Audit**: All communications and actions logged
5. **Flexible**: Departments can verify/acknowledge/opt-out at any time
6. **Privacy Respecting**: No individual names displayed publicly

The system ensures the platform can function effectively while respecting department autonomy and maintaining neutrality.
