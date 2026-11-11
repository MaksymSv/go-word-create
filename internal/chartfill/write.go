package chartfill

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes to a temp file next to target and renames it, so a failed
// write never leaves a half-written workbook.
func WriteFileAtomic(target string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".chartfill-*.xlsx")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if info, err := os.Stat(target); err == nil {
		_ = os.Chmod(tmp.Name(), info.Mode().Perm())
	}
	return os.Rename(tmp.Name(), target)
}
