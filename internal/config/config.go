// Package config loads gh-custom-properties settings from
// $HOME/.gh-custom-properties/config (a simple key=value file), then lets
// environment variables and finally command-line flags override those
// defaults, in that order.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/backup"
)

// Config holds the resolved settings for a run.
type Config struct {
	BackupDir string
	Token     string
	Colors    Palette
}

// Dir returns $HOME/.gh-custom-properties, creating it if it doesn't exist.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".gh-custom-properties"), nil
}

// FilePath returns the path to the config file, without requiring it to exist.
func FilePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config"), nil
}

// Overrides carries the environment/flag values that take precedence over
// the config file. A zero value for a field means "not overridden".
type Overrides struct {
	BackupDir string
	Token     string
}

// EnvOverrides reads GH_CUSTOM_PROPERTIES_BACKUP_DIR and GITHUB_TOKEN.
func EnvOverrides() Overrides {
	return Overrides{
		BackupDir: os.Getenv("GH_CUSTOM_PROPERTIES_BACKUP_DIR"),
		Token:     os.Getenv("GITHUB_TOKEN"),
	}
}

// Load reads the config file (if present) and layers env then flag overrides
// on top, applying built-in defaults for anything still unset.
func Load(env, flags Overrides) (Config, error) {
	cfg := Config{}

	path, err := FilePath()
	if err != nil {
		return Config{}, err
	}
	fileValues, err := readFile(path)
	if err != nil {
		return Config{}, err
	}
	cfg.BackupDir = fileValues["backup_dir"]
	cfg.Token = fileValues["token"]
	cfg.Colors = paletteFromFile(fileValues)

	if env.BackupDir != "" {
		cfg.BackupDir = env.BackupDir
	}
	if env.Token != "" {
		cfg.Token = env.Token
	}
	if flags.BackupDir != "" {
		cfg.BackupDir = flags.BackupDir
	}
	if flags.Token != "" {
		cfg.Token = flags.Token
	}

	if cfg.BackupDir == "" {
		def, err := backup.DefaultDir()
		if err != nil {
			return Config{}, err
		}
		cfg.BackupDir = def
	}
	return cfg, nil
}

// readFile parses a key=value file, one setting per line, blank lines and
// lines starting with '#' ignored. A missing file yields an empty map.
func readFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("open config file %s: %w", path, err)
	}
	defer f.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(f)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: expected key=value, got %q", path, lineNo, line)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}
	return values, nil
}
