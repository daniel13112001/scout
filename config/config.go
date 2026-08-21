package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

//go:embed scoutconfig.toml
var defaultConfig []byte

type Config struct {
	Embedder EmbedderConfig `toml:"embedder"`
	DB       DBConfig       `toml:"db"`
	Index    IndexConfig    `toml:"index"`
}

type EmbedderConfig struct {
	ModelPath      string `toml:"model_path"`
	TokenizerPath  string `toml:"tokenizer_path"`
	OrtLibraryPath string `toml:"ort_library_path"`
	BatchSize      int    `toml:"batch_size"`
}

type DBConfig struct {
	Path string `toml:"path"`
}

type IndexConfig struct {
	MaxFileSizeMB     int      `toml:"max_file_size_mb"`
	AllowedExtensions []string `toml:"allowed_extensions"`
	IgnoreDirs        []string `toml:"ignore_dirs"`
	IgnorePatterns    []string `toml:"ignore_patterns"`

	// IgnoreDirMarkers names files whose presence inside a directory means
	// that whole directory should be skipped, regardless of what the
	// directory itself is named. This is for the case IgnoreDirs' exact
	// name matching can't handle - e.g. a Python virtualenv is always
	// identifiable by a pyvenv.cfg file at its root, but the directory
	// holding it can be named anything ("venv", ".venv", "rl-venv", ...).
	IgnoreDirMarkers []string `toml:"ignore_dir_markers"`
}

// scoutDir returns scout's per-user directory, following each OS's own
// convention for where per-user config lives (e.g. ~/.config on Linux,
// ~/Library/Application Support on macOS, %AppData% on Windows). The
// config file, database, and log file all live here.
func scoutDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolving user config directory: %w", err)
	}
	return filepath.Join(dir, "scout"), nil
}

// Path returns the resolved location of scout's global config file.
func Path() (string, error) {
	dir, err := scoutDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "scoutconfig.toml"), nil
}

// LogPath returns the resolved location of scout's log file.
func LogPath() (string, error) {
	dir, err := scoutDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "scout.log"), nil
}

// OpenLog opens scout's log file for appending, creating scout's directory
// and the file itself if they don't exist yet. The caller is responsible
// for closing it.
func OpenLog() (*os.File, error) {
	path, err := LogPath()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating scout directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}

	return f, nil
}

// Load reads scout's global config for normal runtime use, writing the
// embedded default to disk first if no config file exists yet, and
// resolving DB.Path to an absolute path.
func Load() (*Config, error) {
	cfg, path, err := loadRaw()
	if err != nil {
		return nil, err
	}

	if !filepath.IsAbs(cfg.DB.Path) {
		cfg.DB.Path = filepath.Join(filepath.Dir(path), cfg.DB.Path)
	}

	return cfg, nil
}

// LoadForEdit reads the config file exactly as stored on disk, without
// resolving DB.Path to an absolute path. Use this (with Save) when the
// config is about to be modified and written back - Load's absolute
// DB.Path is only meant for runtime use and would otherwise get baked
// permanently into the file the first time any value is changed.
func LoadForEdit() (*Config, error) {
	cfg, _, err := loadRaw()
	return cfg, err
}

// Save writes cfg back to the config file as TOML. This re-serializes the
// whole file from cfg's current values, so any comments in the existing
// file are lost - hand-editing the file directly is the way to change
// values without disturbing its comments.
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("opening config for write: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return nil
}

func loadRaw() (*Config, string, error) {
	path, err := Path()
	if err != nil {
		return nil, "", err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, "", fmt.Errorf("creating config directory: %w", err)
		}
		if err := os.WriteFile(path, defaultConfig, 0o644); err != nil {
			return nil, "", fmt.Errorf("writing default config: %w", err)
		}
	} else if err != nil {
		return nil, "", fmt.Errorf("checking config file: %w", err)
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, "", fmt.Errorf("parsing config %s: %w", path, err)
	}

	return &cfg, path, nil
}
