// Package backup writes timestamped JSON snapshots of custom-property
// values before they are overwritten, so an operator can recover the
// prior state of a repo (or a whole batch run) after applying a change.
package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
)

// Mode identifies whether a snapshot was taken for a single-repo edit or a
// batch run touching many repos.
type Mode string

const (
	ModeSingle Mode = "single"
	ModeBatch  Mode = "batch"
)

// RepoSnapshot is the pre-change state of one repo's custom properties.
type RepoSnapshot struct {
	Owner            string                   `json:"owner"`
	Repo             string                   `json:"repo"`
	PropertiesBefore []ghclient.PropertyValue `json:"properties_before"`
}

// Snapshot is the full contents of one backup file.
type Snapshot struct {
	Timestamp time.Time      `json:"timestamp"`
	Mode      Mode           `json:"mode"`
	Repos     []RepoSnapshot `json:"repos"`
}

// DefaultDir returns the default backup directory, $HOME/.gh-custom-properties/backups.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".gh-custom-properties", "backups"), nil
}

// Write records a snapshot of the given repos' pre-change property values
// as a single timestamped JSON file under dir, creating dir if needed, and
// returns the path written to.
//
// Callers must call Write with the state fetched immediately before issuing
// any update, and must not issue the update if the fetch failed — a partial
// or missing backup must never be written for a change that goes on to apply.
func Write(dir string, mode Mode, repos []RepoSnapshot) (string, error) {
	if len(repos) == 0 {
		return "", fmt.Errorf("no repos to back up")
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create backup directory %s: %w", dir, err)
	}

	snap := Snapshot{Timestamp: time.Now().UTC(), Mode: mode, Repos: repos}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode backup: %w", err)
	}

	path, err := uniquePath(dir, fileName(mode, repos, snap.Timestamp))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write backup %s: %w", path, err)
	}
	return path, nil
}

// uniquePath appends a numeric suffix to name if a file at dir/name already
// exists, so two applies within the same second don't clobber each other.
func uniquePath(dir, name string) (string, error) {
	path := filepath.Join(dir, name)
	ext := filepath.Ext(name)
	base := name[:len(name)-len(ext)]
	for i := 1; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, nil
		} else if err != nil {
			return "", fmt.Errorf("check backup path %s: %w", path, err)
		}
		path = filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
	}
}

func fileName(mode Mode, repos []RepoSnapshot, ts time.Time) string {
	stamp := ts.Format("20060102-150405")
	if mode == ModeSingle && len(repos) == 1 {
		return fmt.Sprintf("%s-%s-%s.json", repos[0].Owner, repos[0].Repo, stamp)
	}
	return fmt.Sprintf("batch-%s.json", stamp)
}
