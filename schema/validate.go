// Package schema provides startup validation of required DB columns to prevent schema-code mismatch.
package schema

import (
	"database/sql"
	"log"
	"strings"
)

// RequiredColumn defines a required column for a table.
type RequiredColumn struct {
	Table  string
	Column string
}

// DefaultRequiredColumns returns the columns required for escalation and status history to work.
// If any are missing, the server should not start (avoids runtime failures and timezone/actor_type issues).
var DefaultRequiredColumns = []RequiredColumn{
	{Table: "complaint_status_history", Column: "actor_type"},
	{Table: "complaint_status_history", Column: "actor_id"},
	{Table: "complaint_status_history", Column: "reason"},
}

// ValidateRequiredColumns checks that all required columns exist. On failure, logs a fatal error listing missing columns.
func ValidateRequiredColumns(db *sql.DB, required []RequiredColumn) {
	if len(required) == 0 {
		required = DefaultRequiredColumns
	}
	var missing []string
	for _, rc := range required {
		exists, err := columnExists(db, rc.Table, rc.Column)
		if err != nil {
			log.Fatalf("[SCHEMA] Failed to check column %s.%s: %v", rc.Table, rc.Column, err)
		}
		if !exists {
			missing = append(missing, rc.Table+"."+rc.Column)
		}
	}
	if len(missing) > 0 {
		log.Fatalf("[SCHEMA] Missing required columns (run migrations to fix): %s", strings.Join(missing, ", "))
	}
	log.Println("[SCHEMA] Required columns verified")
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.COLUMNS 
		 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
		table, column,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
