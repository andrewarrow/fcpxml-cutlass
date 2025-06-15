package fcp

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
)

// generateUID creates a consistent UID from a file path using MD5 hash
// Uses only the filename to ensure the same file gets the same UID regardless of directory
//
// 🚨 CLAUDE.md Rule: UID Consistency Requirements
// - UIDs MUST be deterministic based on file content/name, not file path
// - Once FCP imports a media file with a specific UID, that UID is permanently associated
// - Different UIDs for same file cause "cannot be imported again with different unique identifier" errors
func generateUID(filePath string) string {
	// Use only the filename (not full path) to ensure consistent UIDs across different working directories
	filename := filepath.Base(filePath)
	hasher := md5.New()
	hasher.Write([]byte("cutlass_video_" + filename))
	hash := hasher.Sum(nil)
	// Convert to uppercase hex string and format as UID
	hexStr := strings.ToUpper(hex.EncodeToString(hash))
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hexStr[0:8], hexStr[8:12], hexStr[12:16], hexStr[16:20], hexStr[20:32])
}

// GenerateUID is the public interface for UID generation
func GenerateUID(filePath string) string {
	return generateUID(filePath)
}

// GenerateTextStyleID creates a unique text style ID based on content and baseName
// CRITICAL: This ensures text-style-def IDs are unique across the entire FCPXML document.
// Never hardcode text style IDs like "ts1" as this causes DTD validation failures
// when multiple text overlays are added to the same project.
func GenerateTextStyleID(text, baseName string) string {
	// Use the existing generateUID function to create a hash-based ID
	fullText := "text_" + baseName + "_" + text
	uid := generateUID(fullText)
	// Return a shorter, more readable ID using the first 8 characters
	return "ts" + uid[0:8]
}

// GenerateResourceID creates a standardized resource ID
func GenerateResourceID(index int) string {
	return fmt.Sprintf("r%d", index)
}

// IDGenerator provides centralized ID generation with collision detection
type IDGenerator struct {
	usedIDs   map[string]bool
	nextIndex int
	fileUIDs  map[string]string // filename -> UID mapping
}

// NewIDGenerator creates a new ID generator
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{
		usedIDs:   make(map[string]bool),
		fileUIDs:  make(map[string]string),
		nextIndex: 1,
	}
}

// ReserveID reserves a single ID and marks it as used
func (g *IDGenerator) ReserveID() string {
	for {
		id := GenerateResourceID(g.nextIndex)
		g.nextIndex++

		if !g.usedIDs[id] {
			g.usedIDs[id] = true
			return id
		}
	}
}

// ReserveIDs reserves multiple IDs in sequence
func (g *IDGenerator) ReserveIDs(count int) []string {
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		ids[i] = g.ReserveID()
	}
	return ids
}

// MarkUsed marks an ID as used (for existing resources)
func (g *IDGenerator) MarkUsed(id string) {
	g.usedIDs[id] = true
}

// IsUsed checks if an ID is already used
func (g *IDGenerator) IsUsed(id string) bool {
	return g.usedIDs[id]
}

// GetConsistentUID returns a consistent UID for a filename
func (g *IDGenerator) GetConsistentUID(filename string) string {
	if uid, exists := g.fileUIDs[filename]; exists {
		return uid
	}

	uid := GenerateUID(filename)
	g.fileUIDs[filename] = uid
	return uid
}
