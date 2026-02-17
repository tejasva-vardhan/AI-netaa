# District-Level Configuration System Design

## Overview

A database-driven configuration system that allows each district to have customized settings for complaint routing, escalation, SLAs, and communication preferences. This design enables the platform to scale to multiple districts while maintaining district-specific operational requirements.

## Target District: Shivpuri, Madhya Pradesh

- **District Name**: Shivpuri
- **State**: Madhya Pradesh
- **PIN Code**: 473551
- **Language**: Hindi (primary), English (secondary)

## Configuration Tables

### 1. district_configurations

**Purpose**: Master configuration table for each district

```sql
CREATE TABLE district_configurations (
    config_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    district_name VARCHAR(255) NOT NULL,
    state VARCHAR(255) NOT NULL,
    pincode VARCHAR(10) NOT NULL,
    location_id BIGINT NOT NULL,
    FOREIGN KEY (location_id) REFERENCES locations(location_id),
    
    -- Language preferences
    primary_language VARCHAR(50) NOT NULL DEFAULT 'hi',
    secondary_languages JSON NULL, -- ['en']
    
    -- Communication preferences
    default_notification_channel ENUM('email', 'sms', 'whatsapp') NOT NULL DEFAULT 'sms',
    office_hours_start TIME NOT NULL DEFAULT '09:00:00',
    office_hours_end TIME NOT NULL DEFAULT '18:00:00',
    working_days JSON NOT NULL, -- ['monday', 'tuesday', 'wednesday', 'thursday', 'friday']
    
    -- Escalation defaults
    default_escalation_time_hours INT NOT NULL DEFAULT 48,
    max_escalation_levels INT NOT NULL DEFAULT 3,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    
    UNIQUE KEY uk_district_location (location_id),
    INDEX idx_state_district (state, district_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 2. district_departments

**Purpose**: Department registry for the district (roles/designations, not person names)

```sql
CREATE TABLE district_departments (
    dept_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    district_config_id BIGINT NOT NULL,
    FOREIGN KEY (district_config_id) REFERENCES district_configurations(config_id),
    
    -- Department info
    department_code VARCHAR(50) NOT NULL,
    department_name VARCHAR(255) NOT NULL,
    department_name_hindi VARCHAR(255) NULL,
    parent_department_id BIGINT NULL,
    FOREIGN KEY (parent_department_id) REFERENCES district_departments(dept_id),
    
    -- Department roles (designations)
    roles JSON NOT NULL, -- ['District Collector', 'Deputy Collector', 'Tehsildar', 'Nayab Tehsildar']
    
    -- Contact (generic, not person-specific)
    contact_email_template VARCHAR(255) NULL, -- 'dept-{code}@shivpuri.mp.gov.in'
    contact_phone VARCHAR(20) NULL,
    
    -- Operational settings
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    priority_order INT NOT NULL DEFAULT 0, -- For display/sorting
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    
    UNIQUE KEY uk_district_dept_code (district_config_id, department_code),
    INDEX idx_district_active (district_config_id, is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3. department_designations

**Purpose**: Officer designations/hierarchy levels within departments

```sql
CREATE TABLE department_designations (
    designation_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    department_id BIGINT NOT NULL,
    FOREIGN KEY (department_id) REFERENCES district_departments(dept_id),
    
    -- Designation details
    designation_code VARCHAR(50) NOT NULL,
    designation_name VARCHAR(255) NOT NULL,
    designation_name_hindi VARCHAR(255) NULL,
    
    -- Hierarchy level (1 = lowest, higher = more senior)
    hierarchy_level INT NOT NULL,
    
    -- Escalation settings
    can_receive_escalations BOOLEAN NOT NULL DEFAULT TRUE,
    can_escalate_to_level INT NULL, -- Next level for escalation
    
    -- Contact template
    email_template VARCHAR(255) NULL, -- '{designation_code}@shivpuri.mp.gov.in'
    
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_dept_designation (department_id, designation_code),
    INDEX idx_hierarchy (department_id, hierarchy_level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 4. category_department_mapping

**Purpose**: Maps complaint categories to responsible departments

```sql
CREATE TABLE category_department_mapping (
    mapping_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    district_config_id BIGINT NOT NULL,
    FOREIGN KEY (district_config_id) REFERENCES district_configurations(config_id),
    
    -- Category mapping
    category VARCHAR(100) NOT NULL,
    primary_department_id BIGINT NOT NULL,
    FOREIGN KEY (primary_department_id) REFERENCES district_departments(dept_id),
    
    -- Secondary departments (for complex issues)
    secondary_department_ids JSON NULL, -- [2, 5] - department IDs
    
    -- Routing rules
    auto_assign BOOLEAN NOT NULL DEFAULT TRUE,
    requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Priority override
    default_priority ENUM('low', 'medium', 'high', 'urgent') NULL,
    
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_district_category (district_config_id, category),
    INDEX idx_department (primary_department_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 5. escalation_hierarchy

**Purpose**: Defines escalation paths within district

```sql
CREATE TABLE escalation_hierarchy (
    hierarchy_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    district_config_id BIGINT NOT NULL,
    FOREIGN KEY (district_config_id) REFERENCES district_configurations(config_id),
    
    -- Escalation level (1, 2, 3...)
    escalation_level INT NOT NULL,
    
    -- From department/designation
    from_department_id BIGINT NULL,
    FOREIGN KEY (from_department_id) REFERENCES district_departments(dept_id),
    from_designation_id BIGINT NULL,
    FOREIGN KEY (from_designation_id) REFERENCES department_designations(designation_id),
    
    -- To department/designation
    to_department_id BIGINT NOT NULL,
    FOREIGN KEY (to_department_id) REFERENCES district_departments(dept_id),
    to_designation_id BIGINT NULL,
    FOREIGN KEY (to_designation_id) REFERENCES department_designations(designation_id),
    
    -- Escalation conditions (JSON)
    conditions JSON NULL, -- {'hours_since_last_update': 48, 'statuses': ['under_review']}
    
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_district_level (district_config_id, escalation_level),
    INDEX idx_from_dept (from_department_id),
    INDEX idx_to_dept (to_department_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 6. department_sla_config

**Purpose**: SLA configuration per department

```sql
CREATE TABLE department_sla_config (
    sla_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    department_id BIGINT NOT NULL,
    FOREIGN KEY (department_id) REFERENCES district_departments(dept_id),
    
    -- SLA by priority
    priority ENUM('low', 'medium', 'high', 'urgent') NOT NULL,
    
    -- Time limits (in hours)
    first_response_hours INT NOT NULL,
    resolution_hours INT NOT NULL,
    
    -- Escalation triggers
    escalation_after_hours INT NULL, -- Escalate if not resolved in X hours
    
    -- Business hours consideration
    count_business_hours_only BOOLEAN NOT NULL DEFAULT FALSE,
    
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_dept_priority (department_id, priority),
    INDEX idx_department (department_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 7. district_escalation_timelines

**Purpose**: District-specific escalation time windows

```sql
CREATE TABLE district_escalation_timelines (
    timeline_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    district_config_id BIGINT NOT NULL,
    FOREIGN KEY (district_config_id) REFERENCES district_configurations(config_id),
    
    -- Timeline configuration
    escalation_level INT NOT NULL,
    hours_since_last_update INT NOT NULL,
    hours_since_status_change INT NULL,
    hours_since_creation INT NULL,
    
    -- Applicable statuses
    applicable_statuses JSON NULL, -- ['verified', 'under_review', 'in_progress']
    
    -- Applicable priorities
    applicable_priorities JSON NULL, -- ['high', 'urgent']
    
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_district_level (district_config_id, escalation_level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## Example Configuration for Shivpuri District

### 1. District Configuration Entry

```sql
INSERT INTO district_configurations (
    district_name, state, pincode, location_id,
    primary_language, secondary_languages,
    default_notification_channel,
    office_hours_start, office_hours_end,
    working_days,
    default_escalation_time_hours,
    max_escalation_levels
) VALUES (
    'Shivpuri',
    'Madhya Pradesh',
    '473551',
    123, -- location_id for Shivpuri (from locations table)
    'hi',
    '["en"]',
    'sms',
    '09:00:00',
    '18:00:00',
    '["monday", "tuesday", "wednesday", "thursday", "friday", "saturday"]',
    48,
    3
);
```

### 2. Department Registry

```sql
-- Public Works Department
INSERT INTO district_departments (
    district_config_id, department_code, department_name,
    department_name_hindi, roles, contact_email_template, contact_phone
) VALUES (
    1, -- district_config_id for Shivpuri
    'PWD',
    'Public Works Department',
    'लोक निर्माण विभाग',
    '["Executive Engineer", "Assistant Engineer", "Junior Engineer", "Supervisor"]',
    'pwd@shivpuri.mp.gov.in',
    '+91-7492-XXXXXX'
);

-- Municipal Corporation
INSERT INTO district_departments (
    district_config_id, department_code, department_name,
    department_name_hindi, roles, contact_email_template, contact_phone
) VALUES (
    1,
    'MC',
    'Municipal Corporation',
    'नगर निगम',
    '["Municipal Commissioner", "Deputy Commissioner", "Sanitation Officer", "Health Officer"]',
    'mc@shivpuri.mp.gov.in',
    '+91-7492-XXXXXX'
);

-- District Collector Office
INSERT INTO district_departments (
    district_config_id, department_code, department_name,
    department_name_hindi, roles, contact_email_template, contact_phone,
    priority_order
) VALUES (
    1,
    'DCO',
    'District Collector Office',
    'जिला कलेक्टर कार्यालय',
    '["District Collector", "Additional Collector", "Deputy Collector", "Tehsildar"]',
    'collector@shivpuri.mp.gov.in',
    '+91-7492-XXXXXX',
    1 -- Highest priority
);

-- Health Department
INSERT INTO district_departments (
    district_config_id, department_code, department_name,
    department_name_hindi, roles, contact_email_template, contact_phone
) VALUES (
    1,
    'HD',
    'Health Department',
    'स्वास्थ्य विभाग',
    '["Chief Medical Officer", "Medical Officer", "Health Inspector"]',
    'health@shivpuri.mp.gov.in',
    '+91-7492-XXXXXX'
);

-- Water Supply Department
INSERT INTO district_departments (
    district_config_id, department_code, department_name,
    department_name_hindi, roles, contact_email_template, contact_phone
) VALUES (
    1,
    'WSD',
    'Water Supply Department',
    'जल आपूर्ति विभाग',
    '["Executive Engineer", "Assistant Engineer", "Water Inspector"]',
    'water@shivpuri.mp.gov.in',
    '+91-7492-XXXXXX'
);

-- Electricity Department
INSERT INTO district_departments (
    district_config_id, department_code, department_name,
    department_name_hindi, roles, contact_email_template, contact_phone
) VALUES (
    1,
    'ED',
    'Electricity Department',
    'बिजली विभाग',
    '["Executive Engineer", "Assistant Engineer", "Lineman"]',
    'electricity@shivpuri.mp.gov.in',
    '+91-7492-XXXXXX'
);
```

### 3. Department Designations (Hierarchy)

```sql
-- District Collector Office Hierarchy
INSERT INTO department_designations (
    department_id, designation_code, designation_name,
    designation_name_hindi, hierarchy_level, can_receive_escalations,
    can_escalate_to_level, email_template
) VALUES
-- Level 1: Tehsildar (lowest)
(3, 'TEHSILDAR', 'Tehsildar', 'तहसीलदार', 1, TRUE, 2, 'tehsildar@shivpuri.mp.gov.in'),
-- Level 2: Deputy Collector
(3, 'DEP_COLL', 'Deputy Collector', 'उप कलेक्टर', 2, TRUE, 3, 'deputy.collector@shivpuri.mp.gov.in'),
-- Level 3: Additional Collector
(3, 'ADD_COLL', 'Additional Collector', 'अतिरिक्त कलेक्टर', 3, TRUE, NULL, 'addl.collector@shivpuri.mp.gov.in'),
-- Level 4: District Collector (highest)
(3, 'DIST_COLL', 'District Collector', 'जिला कलेक्टर', 4, TRUE, NULL, 'collector@shivpuri.mp.gov.in');

-- PWD Hierarchy
INSERT INTO department_designations (
    department_id, designation_code, designation_name,
    designation_name_hindi, hierarchy_level, can_receive_escalations,
    can_escalate_to_level, email_template
) VALUES
(1, 'JE', 'Junior Engineer', 'जूनियर इंजीनियर', 1, TRUE, 2, 'je.pwd@shivpuri.mp.gov.in'),
(1, 'AE', 'Assistant Engineer', 'सहायक इंजीनियर', 2, TRUE, 3, 'ae.pwd@shivpuri.mp.gov.in'),
(1, 'EE', 'Executive Engineer', 'कार्यकारी इंजीनियर', 3, TRUE, NULL, 'ee.pwd@shivpuri.mp.gov.in');

-- Municipal Corporation Hierarchy
INSERT INTO department_designations (
    department_id, designation_code, designation_name,
    designation_name_hindi, hierarchy_level, can_receive_escalations,
    can_escalate_to_level, email_template
) VALUES
(2, 'HO', 'Health Officer', 'स्वास्थ्य अधिकारी', 1, TRUE, 2, 'ho.mc@shivpuri.mp.gov.in'),
(2, 'SO', 'Sanitation Officer', 'स्वच्छता अधिकारी', 1, TRUE, 2, 'so.mc@shivpuri.mp.gov.in'),
(2, 'DEP_COMM', 'Deputy Commissioner', 'उप आयुक्त', 2, TRUE, 3, 'deputy.comm@shivpuri.mp.gov.in'),
(2, 'MUN_COMM', 'Municipal Commissioner', 'नगर आयुक्त', 3, TRUE, NULL, 'commissioner@shivpuri.mp.gov.in');
```

### 4. Category-to-Department Mapping

```sql
-- Infrastructure complaints → PWD
INSERT INTO category_department_mapping (
    district_config_id, category, primary_department_id,
    auto_assign, default_priority
) VALUES (
    1, 'infrastructure', 1, -- PWD
    TRUE, 'medium'
);

-- Sanitation complaints → Municipal Corporation
INSERT INTO category_department_mapping (
    district_config_id, category, primary_department_id,
    auto_assign, default_priority
) VALUES (
    1, 'sanitation', 2, -- MC
    TRUE, 'high'
);

-- Public Safety → Multiple departments (MC primary, PWD secondary)
INSERT INTO category_department_mapping (
    district_config_id, category, primary_department_id,
    secondary_department_ids, auto_assign, default_priority
) VALUES (
    1, 'public_safety', 2, -- MC primary
    '[1]', -- PWD secondary (for street lights, etc.)
    TRUE, 'high'
);

-- Service Delivery → District Collector Office
INSERT INTO category_department_mapping (
    district_config_id, category, primary_department_id,
    auto_assign, requires_approval, default_priority
) VALUES (
    1, 'service_delivery', 3, -- DCO
    TRUE, TRUE, 'medium'
);

-- Water Supply → Water Supply Department
INSERT INTO category_department_mapping (
    district_config_id, category, primary_department_id,
    auto_assign, default_priority
) VALUES (
    1, 'water_supply', 5, -- WSD
    TRUE, 'high'
);

-- Electricity → Electricity Department
INSERT INTO category_department_mapping (
    district_config_id, category, primary_department_id,
    auto_assign, default_priority
) VALUES (
    1, 'electricity', 6, -- ED
    TRUE, 'urgent'
);

-- Environment → Municipal Corporation + Health Department
INSERT INTO category_department_mapping (
    district_config_id, category, primary_department_id,
    secondary_department_ids, auto_assign, default_priority
) VALUES (
    1, 'environment', 2, -- MC primary
    '[4]', -- Health Department secondary
    TRUE, 'medium'
);
```

### 5. Escalation Hierarchy

```sql
-- Level 1: Department internal escalation (e.g., JE → AE → EE in PWD)
INSERT INTO escalation_hierarchy (
    district_config_id, escalation_level,
    from_department_id, from_designation_id,
    to_department_id, to_designation_id,
    conditions
) VALUES (
    1, 1,
    1, -- PWD
    (SELECT designation_id FROM department_designations WHERE designation_code = 'JE' AND department_id = 1),
    1, -- PWD
    (SELECT designation_id FROM department_designations WHERE designation_code = 'AE' AND department_id = 1),
    '{"hours_since_last_update": 48, "statuses": ["under_review", "in_progress"]}'
);

-- Level 2: Cross-department escalation (e.g., PWD → District Collector)
INSERT INTO escalation_hierarchy (
    district_config_id, escalation_level,
    from_department_id,
    to_department_id, to_designation_id,
    conditions
) VALUES (
    1, 2,
    1, -- PWD
    3, -- DCO
    (SELECT designation_id FROM department_designations WHERE designation_code = 'DEP_COLL' AND department_id = 3),
    '{"hours_since_last_update": 72, "statuses": ["in_progress"], "priorities": ["high", "urgent"]}'
);

-- Level 3: Highest escalation (District Collector → Additional Collector)
INSERT INTO escalation_hierarchy (
    district_config_id, escalation_level,
    from_department_id, from_designation_id,
    to_department_id, to_designation_id,
    conditions
) VALUES (
    1, 3,
    3, -- DCO
    (SELECT designation_id FROM department_designations WHERE designation_code = 'DEP_COLL' AND department_id = 3),
    3, -- DCO
    (SELECT designation_id FROM department_designations WHERE designation_code = 'ADD_COLL' AND department_id = 3),
    '{"hours_since_last_update": 120, "statuses": ["in_progress"], "priorities": ["urgent"]}'
);
```

### 6. SLA Configuration

```sql
-- PWD SLAs
INSERT INTO department_sla_config (
    department_id, priority, first_response_hours,
    resolution_hours, escalation_after_hours,
    count_business_hours_only
) VALUES
(1, 'low', 24, 168, 144, FALSE),      -- 1 week resolution
(1, 'medium', 12, 72, 60, FALSE),      -- 3 days resolution
(1, 'high', 6, 48, 36, FALSE),         -- 2 days resolution
(1, 'urgent', 2, 24, 18, FALSE);      -- 1 day resolution

-- Municipal Corporation SLAs
INSERT INTO department_sla_config (
    department_id, priority, first_response_hours,
    resolution_hours, escalation_after_hours,
    count_business_hours_only
) VALUES
(2, 'low', 24, 120, 96, FALSE),
(2, 'medium', 12, 48, 36, FALSE),
(2, 'high', 6, 24, 18, FALSE),
(2, 'urgent', 2, 12, 8, FALSE);

-- District Collector Office SLAs
INSERT INTO department_sla_config (
    department_id, priority, first_response_hours,
    resolution_hours, escalation_after_hours,
    count_business_hours_only
) VALUES
(3, 'low', 48, 240, 192, TRUE),        -- Business hours only
(3, 'medium', 24, 96, 72, TRUE),
(3, 'high', 12, 48, 36, TRUE),
(3, 'urgent', 4, 24, 18, TRUE);

-- Water Supply Department SLAs
INSERT INTO department_sla_config (
    department_id, priority, first_response_hours,
    resolution_hours, escalation_after_hours,
    count_business_hours_only
) VALUES
(5, 'low', 24, 72, 60, FALSE),
(5, 'medium', 12, 48, 36, FALSE),
(5, 'high', 6, 24, 18, FALSE),
(5, 'urgent', 2, 12, 8, FALSE);

-- Electricity Department SLAs
INSERT INTO department_sla_config (
    department_id, priority, first_response_hours,
    resolution_hours, escalation_after_hours,
    count_business_hours_only
) VALUES
(6, 'low', 24, 96, 72, FALSE),
(6, 'medium', 12, 48, 36, FALSE),
(6, 'high', 6, 24, 18, FALSE),
(6, 'urgent', 1, 6, 4, FALSE);        -- Very fast for urgent electricity issues
```

### 7. District Escalation Timelines

```sql
-- Level 1 Escalation: After 48 hours
INSERT INTO district_escalation_timelines (
    district_config_id, escalation_level,
    hours_since_last_update,
    applicable_statuses, applicable_priorities
) VALUES (
    1, 1,
    48,
    '["verified", "under_review", "in_progress"]',
    '["medium", "high", "urgent"]'
);

-- Level 2 Escalation: After 72 hours (for high priority)
INSERT INTO district_escalation_timelines (
    district_config_id, escalation_level,
    hours_since_last_update,
    applicable_statuses, applicable_priorities
) VALUES (
    1, 2,
    72,
    '["in_progress"]',
    '["high", "urgent"]'
);

-- Level 3 Escalation: After 120 hours (for urgent only)
INSERT INTO district_escalation_timelines (
    district_config_id, escalation_level,
    hours_since_last_update,
    applicable_statuses, applicable_priorities
) VALUES (
    1, 3,
    120,
    '["in_progress"]',
    '["urgent"]'
);
```

## How Configuration Controls Complaint Routing

### 1. Initial Complaint Assignment

**Flow**:
```
User files complaint with category "infrastructure"
  ↓
System queries category_department_mapping
  WHERE district_config_id = 1 AND category = 'infrastructure'
  ↓
Result: primary_department_id = 1 (PWD)
  ↓
System queries district_departments
  WHERE dept_id = 1
  ↓
Result: PWD department details
  ↓
System queries department_designations
  WHERE department_id = 1 AND hierarchy_level = 1 (lowest level)
  ↓
Result: Junior Engineer (JE)
  ↓
Complaint assigned to PWD → Junior Engineer
  ↓
Default priority set to "medium" (from category_department_mapping)
```

### 2. Escalation Trigger

**Flow**:
```
Complaint status: "in_progress"
Last update: 50 hours ago
Priority: "high"
  ↓
System queries district_escalation_timelines
  WHERE district_config_id = 1
    AND escalation_level = 1
    AND hours_since_last_update <= 50
    AND 'in_progress' IN applicable_statuses
    AND 'high' IN applicable_priorities
  ↓
Match found: Level 1 escalation triggered
  ↓
System queries escalation_hierarchy
  WHERE district_config_id = 1
    AND escalation_level = 1
    AND from_department_id = 1 (PWD)
  ↓
Result: Escalate to Assistant Engineer (AE) in PWD
  ↓
System updates complaint:
  - Status: "escalated"
  - Assigned to: AE
  - Escalation level: 1
  ↓
Create escalation record in complaint_escalations
```

### 3. SLA Monitoring

**Flow**:
```
Complaint created: 2026-02-10 10:00:00
Priority: "high"
Department: PWD
  ↓
System queries department_sla_config
  WHERE department_id = 1 AND priority = 'high'
  ↓
Result:
  - first_response_hours: 6
  - resolution_hours: 48
  - escalation_after_hours: 36
  ↓
Current time: 2026-02-10 18:00:00 (8 hours later)
  ↓
Check: 8 hours > 6 hours (first_response_hours)
  ↓
Trigger: First response SLA breached
  ↓
Action: Send reminder notification to assigned officer
```

### 4. Multi-Department Routing

**Flow**:
```
Complaint category: "public_safety"
  ↓
System queries category_department_mapping
  WHERE category = 'public_safety'
  ↓
Result:
  - primary_department_id: 2 (MC)
  - secondary_department_ids: [1] (PWD)
  ↓
Complaint assigned to MC (primary)
  ↓
System creates notification for PWD (secondary)
  ↓
Both departments can view and collaborate
```

## Configuration Benefits

### 1. Scalability

- **Add New Districts**: Insert new row in `district_configurations`
- **District-Specific Rules**: Each district has independent configuration
- **No Code Changes**: All routing logic driven by database

### 2. Flexibility

- **Department Changes**: Update `district_departments` without code changes
- **Escalation Rules**: Modify `escalation_hierarchy` for different escalation paths
- **SLA Adjustments**: Update `department_sla_config` per department needs

### 3. Maintainability

- **Centralized Configuration**: All district settings in database
- **Audit Trail**: Track configuration changes via `updated_at` timestamps
- **Version Control**: Can add versioning to configurations if needed

### 4. Localization

- **Language Support**: `primary_language` and `secondary_languages` fields
- **Hindi Names**: `department_name_hindi` and `designation_name_hindi`
- **Communication**: `default_notification_channel` for preferred communication

## Query Examples

### Get Department for Category

```sql
SELECT d.*
FROM category_department_mapping m
JOIN district_departments d ON m.primary_department_id = d.dept_id
WHERE m.district_config_id = 1
  AND m.category = 'infrastructure'
  AND m.is_active = TRUE;
```

### Get Escalation Path

```sql
SELECT eh.*,
       from_dept.department_name as from_department,
       to_dept.department_name as to_department,
       from_des.designation_name as from_designation,
       to_des.designation_name as to_designation
FROM escalation_hierarchy eh
JOIN district_departments from_dept ON eh.from_department_id = from_dept.dept_id
JOIN district_departments to_dept ON eh.to_department_id = to_dept.dept_id
LEFT JOIN department_designations from_des ON eh.from_designation_id = from_des.designation_id
LEFT JOIN department_designations to_des ON eh.to_designation_id = to_des.designation_id
WHERE eh.district_config_id = 1
  AND eh.escalation_level = 1
  AND eh.is_active = TRUE
ORDER BY eh.escalation_level;
```

### Get SLA for Department and Priority

```sql
SELECT *
FROM department_sla_config
WHERE department_id = 1
  AND priority = 'high'
  AND is_active = TRUE;
```

### Get Next Escalation Level

```sql
SELECT eh.*
FROM escalation_hierarchy eh
WHERE eh.district_config_id = 1
  AND eh.escalation_level = (
    SELECT COALESCE(MAX(escalation_level), 0) + 1
    FROM complaint_escalations
    WHERE complaint_id = ?
  )
  AND eh.from_department_id = ?
  AND eh.is_active = TRUE
ORDER BY eh.escalation_level
LIMIT 1;
```

## Summary

This configuration system provides:

1. **District-Specific Settings**: Each district has independent configuration
2. **Department Registry**: Roles/designations without person names
3. **Category Routing**: Automatic department assignment based on category
4. **Escalation Hierarchy**: Multi-level escalation paths
5. **SLA Management**: Department and priority-specific SLAs
6. **Language Support**: Hindi and English preferences
7. **Communication Preferences**: Default notification channels

All routing, escalation, and SLA logic is driven by database configuration, making it easy to add new districts and modify existing ones without code changes.
