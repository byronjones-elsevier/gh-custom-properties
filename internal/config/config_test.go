package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfigFile(t *testing.T, home, contents string) {
	t.Helper()
	dir := filepath.Join(home, ".gh-custom-properties")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestLoad_Precedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfigFile(t, home, "backup_dir=/from-file\ntoken=file-token\n")

	t.Run("file only", func(t *testing.T) {
		cfg, err := Load(Overrides{}, Overrides{})
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.BackupDir != "/from-file" || cfg.Token != "file-token" {
			t.Errorf("cfg = %+v, want file values", cfg)
		}
	})

	t.Run("env overrides file", func(t *testing.T) {
		env := Overrides{BackupDir: "/from-env", Token: "env-token"}
		cfg, err := Load(env, Overrides{})
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.BackupDir != "/from-env" || cfg.Token != "env-token" {
			t.Errorf("cfg = %+v, want env values", cfg)
		}
	})

	t.Run("flag overrides env and file", func(t *testing.T) {
		env := Overrides{BackupDir: "/from-env", Token: "env-token"}
		flags := Overrides{BackupDir: "/from-flag", Token: "flag-token"}
		cfg, err := Load(env, flags)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.BackupDir != "/from-flag" || cfg.Token != "flag-token" {
			t.Errorf("cfg = %+v, want flag values", cfg)
		}
	})
}

func TestLoad_NoFile_UsesDefaultBackupDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := Load(Overrides{}, Overrides{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := filepath.Join(home, ".gh-custom-properties", "backups")
	if cfg.BackupDir != want {
		t.Errorf("cfg.BackupDir = %q, want %q", cfg.BackupDir, want)
	}
}

func TestLoad_MalformedLine(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfigFile(t, home, "not-a-key-value-line\n")

	if _, err := Load(Overrides{}, Overrides{}); err == nil {
		t.Fatal("expected an error for a malformed config line")
	}
}
