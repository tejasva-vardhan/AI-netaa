package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"time"
)

// GenerateEvidenceHash generates SHA-256 hash for evidence integrity.
//
// Hash input is RAW BYTES only (no hex string, no URL-derived content):
//   image_bytes (raw) || latitude (float64 LE) || longitude (float64 LE) || captured_at (Unix nano int64 LE)
//
// Timestamp MUST be server-generated at upload time. Do not use client-provided timestamps.
//
// Trust model: The evidence hash is an INTEGRITY SIGNAL (detects tampering after capture).
// It is NOT authenticity proof (does not prove who/when/where beyond what we record).
// Do not overclaim in legal or compliance contexts.
func GenerateEvidenceHash(
	imageBytes []byte,
	latitude float64,
	longitude float64,
	capturedAt time.Time, // Server-generated timestamp only
) string {
	buf := bytes.NewBuffer(imageBytes)

	// Append latitude as float64 little-endian
	_ = binary.Write(buf, binary.LittleEndian, latitude)
	// Append longitude as float64 little-endian
	_ = binary.Write(buf, binary.LittleEndian, longitude)
	// Append server timestamp as Unix nanoseconds (int64 little-endian)
	_ = binary.Write(buf, binary.LittleEndian, capturedAt.UnixNano())

	hash := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(hash[:])
}
