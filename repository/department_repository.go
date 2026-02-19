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
// For pilot: Uses simple rule-based mapping (category → department_id, then location → SDM as fallback)
// In production: Query category_department_mapping table
func (r *DepartmentRepository) GetDepartmentByCategoryAndLocation(
	category string,
	locationID int64,
) (*int64, *string, error) {
	// For pilot: Simple rule-based mapping
	// Priority 1: Category-based mapping (PRIMARY)
	// Priority 2: Location-based fallback (ONLY if category missing/fails)
	// Priority 3: Default (District Collector Office)
	
	var departmentID int64
	var priorityOverride *string
	
	// PRIMARY: Try category-based mapping first
	if category != "" {
		switch category {
		case "infrastructure":
			departmentID = 1 // PWD (Public Works Department)
			priority := "medium"
			priorityOverride = &priority
			return &departmentID, priorityOverride, nil
		case "water":
			departmentID = 2 // Water Supply Department
			priority := "high"
			priorityOverride = &priority
			return &departmentID, priorityOverride, nil
		case "electricity":
			departmentID = 3 // Electricity Department
			priority := "urgent"
			priorityOverride = &priority
			return &departmentID, priorityOverride, nil
		case "sanitation":
			departmentID = 4 // Municipal Corporation
			priority := "high"
			priorityOverride = &priority
			return &departmentID, priorityOverride, nil
		case "health":
			departmentID = 5 // Health Department
			priority := "high"
			priorityOverride = &priority
			return &departmentID, priorityOverride, nil
		case "education":
			departmentID = 6 // Education Department
			priority := "medium"
			priorityOverride = &priority
			return &departmentID, priorityOverride, nil
		}
		// If category exists but doesn't match any case, fall through to location check
	}
	
	// FALLBACK: Location-based routing (ONLY if category missing or doesn't match)
	// Pilot hardcoding: Location IDs for Kolaras, Pohri, Karera
	// In production: Query locations table to get actual location IDs
	const (
		locationKolaras int64 = 2 // Kolaras tehsil
		locationPohri   int64 = 3 // Pohri tehsil
		locationKarera  int64 = 4 // Karera tehsil
	)
	
	// SDM Department IDs (pilot hardcoding)
	const (
		deptSDMKolaras int64 = 8 // SDM Kolaras
		deptSDMPohri   int64 = 9 // SDM Pohri
		deptSDMKarera  int64 = 10 // SDM Karera
	)
	
	switch locationID {
	case locationKolaras:
		departmentID = deptSDMKolaras
		priority := "high"
		priorityOverride = &priority
		return &departmentID, priorityOverride, nil
	case locationPohri:
		departmentID = deptSDMPohri
		priority := "high"
		priorityOverride = &priority
		return &departmentID, priorityOverride, nil
	case locationKarera:
		departmentID = deptSDMKarera
		priority := "high"
		priorityOverride = &priority
		return &departmentID, priorityOverride, nil
	}
	
	// FINAL FALLBACK: Default (District Collector Office)
	departmentID = 7
	priority := "medium"
	priorityOverride = &priority
	
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

// GetDepartmentName gets department name by ID
// Returns department name or fallback name if not found
func (r *DepartmentRepository) GetDepartmentName(departmentID int64) (string, error) {
	query := `
		SELECT name
		FROM departments
		WHERE department_id = ? AND is_active = true
		LIMIT 1
	`
	
	var name string
	err := r.db.QueryRow(query, departmentID).Scan(&name)
	if err == sql.ErrNoRows {
		// Fallback: return descriptive name based on ID (for pilot)
		return getFallbackDepartmentName(departmentID), nil
	}
	if err != nil {
		return getFallbackDepartmentName(departmentID), fmt.Errorf("failed to get department name: %w", err)
	}
	
	return name, nil
}

// getFallbackDepartmentName returns a descriptive name for known department IDs (pilot fallback)
func getFallbackDepartmentName(departmentID int64) string {
	switch departmentID {
	case 1:
		return "PWD (Public Works Department)"
	case 2:
		return "Water Supply Department"
	case 3:
		return "Electricity Department"
	case 4:
		return "Municipal Corporation"
	case 5:
		return "Health Department"
	case 6:
		return "Education Department"
	case 7:
		return "District Collector Office"
	case 8:
		return "SDM Kolaras"
	case 9:
		return "SDM Pohri"
	case 10:
		return "SDM Karera"
	default:
		return fmt.Sprintf("Department %d", departmentID)
	}
}
