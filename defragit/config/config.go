package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Repo     RepoConfig     `yaml:"repo"`
	User     UserConfig     `yaml:"user"`
	DB       DBConfig       `yaml:"db"`
	Branch   BranchConfig   `yaml:"branch"`
	Identity IdentityConfig `yaml:"identity"`
}

type RepoConfig struct {
	Name        string `yaml:"name"`
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
}

type UserConfig struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type DBConfig struct {
	Path    string `yaml:"path"`
	Port    int    `yaml:"port"`
	P2PPort int    `yaml:"p2p_port"`
}

type BranchConfig struct {
	Default string `yaml:"default"`
	Current string `yaml:"current"`
}

type IdentityConfig struct {
	KeyPath string `yaml:"key_path"`
	PeerID  string `yaml:"peer_id"`
}

// Home returns the root ~/.defragit directory.
func Home() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".defragit")
}

// RepoDir returns the directory for a named repo.
func RepoDir(name string) string {
	return filepath.Join(Home(), name)
}

// Path returns the config file path for a repo.
func Path(repoName string) string {
	return filepath.Join(RepoDir(repoName), "config.yaml")
}

// GlobalPath returns the global config path.
func GlobalPath() string {
	return filepath.Join(Home(), "config.yaml")
}

// IdentityKeyPath returns the shared identity key path.
func IdentityKeyPath() string {
	return filepath.Join(Home(), "identity.key")
}

// Load reads config for the given repo.
func Load(repoName string) (*Config, error) {
	data, err := os.ReadFile(Path(repoName))
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes config to the repo's config file.
func Save(repoName string, cfg *Config) error {
	dir := RepoDir(repoName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating repo dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(Path(repoName), data, 0644)
}

// LoadGlobal reads the global user config (tolerates missing file).
func LoadGlobal() *Config {
	data, err := os.ReadFile(GlobalPath())
	if err != nil {
		return &Config{}
	}
	var cfg Config
	_ = yaml.Unmarshal(data, &cfg)
	return &cfg
}

// SaveGlobal writes the global config.
func SaveGlobal(cfg *Config) error {
	if err := os.MkdirAll(Home(), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(GlobalPath(), data, 0644)
}

// DefaultDBPath returns the default BadgerDB path for a repo.
func DefaultDBPath(repoName string) string {
	return filepath.Join(RepoDir(repoName), "db")
}
