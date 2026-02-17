-- ============================================================================
-- PILOT DATA SEED: Initial Authority Data for Shivpuri, Madhya Pradesh
-- Pincode: 473551
-- ============================================================================
-- 
-- IMPORTANT NOTES:
-- 1. This is PILOT DATA with PLACEHOLDER EMAILS
-- 2. All emails are clearly marked as placeholders (e.g., @gov.in)
-- 3. Data will be manually verified before production use
-- 4. Officer names are generic/representative, not actual individuals
-- 5. Location ID 1 is assumed to be Shivpuri (create if not exists)
--
-- ============================================================================

-- Step 1: Ensure Shivpuri location exists
-- PILOT DATA: Create Shivpuri location if not exists
INSERT INTO locations (location_id, location_type, name, code, is_active, created_at) VALUES
(1, 'district', 'Shivpuri', '473551', TRUE, NOW())
ON DUPLICATE KEY UPDATE name = 'Shivpuri', code = '473551', is_active = TRUE;

-- Step 2: Insert Departments
-- PILOT DATA (PLACEHOLDER EMAILS)

INSERT INTO departments (department_id, name, code, description, is_active, created_at) VALUES
(1, 'Public Works Department', 'PWD', 'Handles infrastructure, roads, and public works', TRUE, NOW()),
(2, 'Water Supply Department', 'PHED', 'Public Health Engineering Department - Water supply and sanitation', TRUE, NOW()),
(3, 'Electricity Distribution', 'DISCOM', 'Electricity distribution and power supply', TRUE, NOW()),
(4, 'Municipal Corporation', 'MUNICIPAL', 'Municipal services, sanitation, and civic amenities', TRUE, NOW()),
(5, 'Health Department', 'HEALTH', 'Public health services and medical facilities', TRUE, NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), code = VALUES(code);

-- Step 3: Insert Officers (3 levels per department = 15 officers total)
-- PILOT DATA (PLACEHOLDER EMAILS)

-- Department 1: Public Works Department (PWD)
INSERT INTO officers (officer_id, employee_id, full_name, designation, email, department_id, location_id, is_active, created_at) VALUES
-- L1: Local Officer
(1, 'PWD-L1-001', 'Shri Rajesh Kumar', 'Junior Engineer', 'pwd.shivpuri.l1@gov.in', 1, 1, TRUE, NOW()),
-- L2: Department Head (City/Block)
(2, 'PWD-L2-001', 'Shri Mahesh Sharma', 'Executive Engineer', 'pwd.shivpuri.l2@gov.in', 1, 1, TRUE, NOW()),
-- L3: District-level Officer
(3, 'PWD-L3-001', 'Shri Anil Verma', 'Superintending Engineer', 'pwd.shivpuri.l3@gov.in', 1, 1, TRUE, NOW()),

-- Department 2: Water Supply Department (PHED)
-- L1: Local Officer
(4, 'PHED-L1-001', 'Shri Vikas Singh', 'Assistant Engineer', 'phed.shivpuri.l1@gov.in', 2, 1, TRUE, NOW()),
-- L2: Department Head (City/Block)
(5, 'PHED-L2-001', 'Shri Sunil Patel', 'Executive Engineer', 'phed.shivpuri.l2@gov.in', 2, 1, TRUE, NOW()),
-- L3: District-level Officer
(6, 'PHED-L3-001', 'Shri Ramesh Yadav', 'Superintending Engineer', 'phed.shivpuri.l3@gov.in', 2, 1, TRUE, NOW()),

-- Department 3: Electricity Distribution (DISCOM)
-- L1: Local Officer
(7, 'DISCOM-L1-001', 'Shri Amit Kumar', 'Junior Engineer', 'discom.shivpuri.l1@gov.in', 3, 1, TRUE, NOW()),
-- L2: Department Head (City/Block)
(8, 'DISCOM-L2-001', 'Shri Pradeep Sharma', 'Assistant Engineer', 'discom.shivpuri.l2@gov.in', 3, 1, TRUE, NOW()),
-- L3: District-level Officer
(9, 'DISCOM-L3-001', 'Shri Sanjay Meena', 'Executive Engineer', 'discom.shivpuri.l3@gov.in', 3, 1, TRUE, NOW()),

-- Department 4: Municipal Corporation
-- L1: Local Officer
(10, 'MUNICIPAL-L1-001', 'Shri Deepak Jain', 'Sanitation Inspector', 'municipal.shivpuri.l1@gov.in', 4, 1, TRUE, NOW()),
-- L2: Department Head (City/Block)
(11, 'MUNICIPAL-L2-001', 'Shri Manoj Gupta', 'Municipal Commissioner', 'municipal.shivpuri.l2@gov.in', 4, 1, TRUE, NOW()),
-- L3: District-level Officer
(12, 'MUNICIPAL-L3-001', 'Shri Ashok Tiwari', 'Additional Commissioner', 'municipal.shivpuri.l3@gov.in', 4, 1, TRUE, NOW()),

-- Department 5: Health Department
-- L1: Local Officer
(13, 'HEALTH-L1-001', 'Dr. Priya Sharma', 'Medical Officer', 'health.shivpuri.l1@gov.in', 5, 1, TRUE, NOW()),
-- L2: Department Head (City/Block)
(14, 'HEALTH-L2-001', 'Dr. Ravi Kumar', 'Chief Medical Officer', 'health.shivpuri.l2@gov.in', 5, 1, TRUE, NOW()),
-- L3: District-level Officer
(15, 'HEALTH-L3-001', 'Dr. Suresh Mehta', 'District Health Officer', 'health.shivpuri.l3@gov.in', 5, 1, TRUE, NOW())
ON DUPLICATE KEY UPDATE full_name = VALUES(full_name), designation = VALUES(designation), email = VALUES(email);

-- Step 4: Department-Location Mapping (if department_location_mapping table exists)
-- Maps departments to Shivpuri location
-- PILOT DATA

INSERT INTO department_location_mapping (department_id, location_id, is_active, created_at) VALUES
(1, 1, TRUE, NOW()), -- PWD → Shivpuri
(2, 1, TRUE, NOW()), -- PHED → Shivpuri
(3, 1, TRUE, NOW()), -- DISCOM → Shivpuri
(4, 1, TRUE, NOW()), -- Municipal → Shivpuri
(5, 1, TRUE, NOW())  -- Health → Shivpuri
ON DUPLICATE KEY UPDATE is_active = TRUE;

-- ============================================================================
-- VERIFICATION QUERIES (Run after seeding to verify data)
-- ============================================================================

-- Verify departments
-- SELECT department_id, name, code FROM departments WHERE is_active = TRUE ORDER BY department_id;

-- Verify officers by department
-- SELECT 
--     o.officer_id,
--     o.full_name,
--     o.designation,
--     o.email,
--     d.name AS department_name,
--     o.is_active
-- FROM officers o
-- JOIN departments d ON o.department_id = d.department_id
-- WHERE o.location_id = 1 AND o.is_active = TRUE
-- ORDER BY d.department_id, o.officer_id;

-- Verify department-location mappings
-- SELECT 
--     dlm.mapping_id,
--     d.name AS department_name,
--     dlm.location_id,
--     dlm.is_active
-- FROM department_location_mapping dlm
-- JOIN departments d ON dlm.department_id = d.department_id
-- WHERE dlm.location_id = 1 AND dlm.is_active = TRUE;

-- ============================================================================
-- END OF SEED DATA
-- ============================================================================
