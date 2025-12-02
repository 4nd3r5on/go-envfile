package common

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreateBackup creates a timestamped backup of the file
func CreateBackup(logger *slog.Logger, path string) error {
	input, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No backup needed for non-existent file
		}
		return fmt.Errorf("failed to read file for backup: %w", err)
	}

	// Generate timestamp-based backup path
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(path)
	basePath := strings.TrimSuffix(path, ext)
	backupPath := fmt.Sprintf("%s.%s.bak%s", basePath, timestamp, ext)

	if err := os.WriteFile(backupPath, input, 0o644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	logger.Info("created backup", "backup_path", backupPath)
	return nil
}
