package repository

import (
	"database/sql"
	"fmt"
)

// DepartmentRepository handles database operations for department routing
type DepartmentRepository struct {
	db *sql.DB
}

// NewDepartmentRepository creates a new department repository
func NewDepartmentRepository(db *sql.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

// GetDepartmentByCategoryAndLocation gets department assignment for a category and location
// Returns department_id and priority override if mapping exists
// For pilot: Uses simple rule-based mapping (category → department_id)
// In production: Query category_department_mapping table
func (r *DepartmentRepository) GetDepartmentByCategoryAndLocation(
	category string,
	locationID int64,
) (*int64, *string, error) {
	// For pilot: Simple rule-based mapping
	// Map category to department_id (default department IDs for Shivpuri)
	// In production, this would query category_department_mapping table
	
	var departmentID int64
	var priorityOverride *string
	
	switch category {
	case "infrastructure":
		departmentID = 1 // PWD (Public Works Department)
		priority := "medium"
		priorityOverride = &priority
	case "water":
		departmentID = 2 // Water Supply Department
		priority := "high"
		priorityOverride = &priority
	case "electricity":
		departmentID = 3 // Electricity Department
		priority := "urgent"
		priorityOverride = &priority
	case "sanitation":
		departmentID = 4 // Municipal Corporation
		priority := "high"
		priorityOverride = &priority
	case "health":
		departmentID = 5 // Health Department
		priority := "high"
		priorityOverride = &priority
	case "education":
		departmentID = 6 // Education Department
		priority := "medium"
		priorityOverride = &priority
	default:
		// General complaints → District Collector Office
		departmentID = 7
		priority := "medium"
		priorityOverride = &priority
	}
	
	// Verify department exists (for pilot, assume it exists)
	// In production: SELECT department_id FROM departments WHERE department_id = ? AND is_active = true
	
	return &departmentID, priorityOverride, nil
}

// FindOfficerForDepartment finds an officer for a department and location
// Returns first available officer or nil if none found
func (r *DepartmentRepository) FindOfficerForDepartment(
	departmentID int64,
	locationID int64,
) (*int64, error) {
	query := `
		SELECT officer_id
		FROM officers
		WHERE department_id = ?
			AND location_id = ?
			AND is_active = true
		LIMIT 1
	`
	
	var officerID sql.NullInt64
	err := r.db.QueryRow(query, departmentID, locationID).Scan(&officerID)
	if err == sql.ErrNoRows {
		// No officer found - that's OK, complaint can be assigned to department only
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find officer: %w", err)
	}
	
	if !officerID.Valid {
		return nil, nil
	}
	
	return &officerID.Int64, nil
}
