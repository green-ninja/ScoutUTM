package ids

import (
	"fmt"
	"log"
	"os"
)

// RotateEVELog rotates the EVE log file if it exceeds maxBytes.
// Keeps at most maxFiles rotated copies (.1 through .N), deleting the oldest.
func RotateEVELog(path string, maxBytes int64, maxFiles int) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil // File doesn't exist yet, nothing to rotate.
	}
	if info.Size() < maxBytes {
		return nil
	}

	// Delete oldest rotated file if at limit.
	oldest := fmt.Sprintf("%s.%d", path, maxFiles)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: removing oldest EVE log %s: %v", oldest, err)
	}

	// Shift existing rotated files: .6 → .7, .5 → .6, etc.
	for i := maxFiles - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", path, i)
		dst := fmt.Sprintf("%s.%d", path, i+1)
		if err := os.Rename(src, dst); err != nil && !os.IsNotExist(err) {
			log.Printf("warning: rotating EVE log %s → %s: %v", src, dst, err)
		}
	}

	// Rotate current file to .1.
	if err := os.Rename(path, fmt.Sprintf("%s.1", path)); err != nil {
		return fmt.Errorf("rotating EVE log: %w", err)
	}

	return nil
}
